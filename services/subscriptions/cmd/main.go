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
	chimw "github.com/go-chi/chi/v5/middleware"

	"payment-processor/internal/config"
	"payment-processor/internal/database"
	"payment-processor/services/subscriptions/handler"
	"payment-processor/services/subscriptions/service"
	"payment-processor/services/subscriptions/worker"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect Database
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Services
	subSvc := service.NewSubscriptionService(db)
	subHandler := handler.NewSubscriptionHandler(subSvc)

	// Start Billing Worker (checks every 10 seconds for demo, usually hourly/daily)
	// We point the worker to the gateway via NGINX port 80
	billingWorker := worker.NewBillingWorker(db, "http://localhost/v1/payments")
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go billingWorker.Start(workerCtx, 10*time.Second)

	// Setup Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Post("/v1/subscriptions", subHandler.CreateSubscription)

	// Start Server
	server := &http.Server{
		Addr:    ":8083",
		Handler: r,
	}

	go func() {
		log.Println("Subscriptions service starting on port 8083...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Subscriptions service...")
	workerCancel()

	ctxShutDown, cancelShutDown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutDown()

	if err := server.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Subscriptions service exited cleanly")
}
