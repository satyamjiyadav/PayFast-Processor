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
	"payment-processor/services/ledger/service"

	kafka_go "github.com/segmentio/kafka-go"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Database
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Kafka Producer (for Webhooks)
	producer := kafka.NewProducer([]string{cfg.KafkaBrokers}, "webhook_events")
	defer producer.Close()

	// Initialize Ledger Service
	ledgerSvc := service.NewLedgerService(db, producer)

	// Initialize Kafka Consumer
	consumer := kafka.NewConsumer([]string{cfg.KafkaBrokers}, "ledger-service-group", "payment_events")
	defer consumer.Close()

	// Start consuming events
	go func() {
		log.Println("Ledger service starting... listening for payment events")
		consumer.Consume(ctx, func(msg kafka_go.Message) error {
			return ledgerSvc.HandlePaymentEvent(msg.Value)
		})
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Ledger service...")
	cancel() // Stop consumer context

	// Give a moment for the current message processing to complete
	time.Sleep(2 * time.Second)
	log.Println("Ledger service exited cleanly")
}
