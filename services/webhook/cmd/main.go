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
	"payment-processor/services/webhook/service"

	kafka_go "github.com/segmentio/kafka-go"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Database (needed to fetch webhook configs)
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Webhook Service
	webhookSvc := service.NewWebhookService(db)

	// Initialize Kafka Consumer for Webhook Events
	consumer := kafka.NewConsumer([]string{cfg.KafkaBrokers}, "webhook-service-group", "webhook_events")
	defer consumer.Close()

	// Start consuming events
	go func() {
		log.Println("Webhook Dispatcher service starting... listening for webhook_events")
		consumer.Consume(ctx, func(msg kafka_go.Message) error {
			return webhookSvc.HandleWebhookEvent(msg.Value)
		})
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Webhook Dispatcher service...")
	cancel() // Stop consumer context

	// Give a moment for the current message processing to complete
	time.Sleep(2 * time.Second)
	log.Println("Webhook Dispatcher exited cleanly")
}
