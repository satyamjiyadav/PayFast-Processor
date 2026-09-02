package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"payment-processor/internal/config"
	"payment-processor/internal/database"
	"payment-processor/internal/kafka"
	"payment-processor/services/settlements/service"
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

	// Initialize Kafka Producer
	producer := kafka.NewProducer([]string{cfg.KafkaBrokers}, "payment_events")
	defer producer.Close()

	// 3. Settlement Service
	svc := service.NewSettlementService(db, producer)

	// 4. Kafka Consumer for Settling Events
	consumer := kafka.NewConsumer([]string{cfg.KafkaBrokers}, "settlement-group", "payment_events")
	defer consumer.Close()

	// 5. Start Consumer
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	go consumer.Consume(consumerCtx, svc.HandleSettleEvent)

	// 6. Start Scheduler
	go svc.StartScheduler(consumerCtx)

	log.Println("Settlement Service is running...")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Settlement Service...")
	consumerCancel()
}
