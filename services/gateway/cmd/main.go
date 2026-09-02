package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"payment-processor/internal/config"
	"payment-processor/internal/database"
	"payment-processor/internal/kafka"
	"payment-processor/internal/redis"
	gatewayMiddleware "payment-processor/services/gateway/middleware"
	"payment-processor/services/gateway/handler"
	"payment-processor/services/gateway/service"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to Database
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	rdb, err := redis.Connect(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer rdb.Close()

	// Initialize Kafka Producer
	producer := kafka.NewProducer([]string{cfg.KafkaBrokers}, "payment_events")
	defer producer.Close()

	// Initialize Services & Handlers
	paymentSvc := service.NewPaymentService(db, producer)
	idempotencySvc := service.NewIdempotencyService(db, rdb)
	paymentHandler := handler.NewPaymentHandler(paymentSvc)

	// Setup Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API Routes with Middlewares
	r.Group(func(r chi.Router) {
		r.Use(gatewayMiddleware.APIAuth(db))
		r.Use(gatewayMiddleware.Idempotency(idempotencySvc))

		r.Post("/v1/payments", paymentHandler.CreatePayment)
	})

	// Start Server
	server := &http.Server{
		Addr:    ":" + cfg.Port, // Defaults to 8080
		Handler: r,
	}

	go func() {
		log.Printf("Gateway service starting on port %s...", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Gateway service...")
	ctxShutDown, cancelShutDown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutDown()

	if err := server.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Gateway service exited cleanly")
}
