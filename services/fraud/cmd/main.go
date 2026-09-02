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
	"payment-processor/internal/redis"
	"payment-processor/services/fraud/service"
	segmentioKafka "github.com/segmentio/kafka-go"
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

	// Initialize Services
	fraudSvc := service.NewFraudService(db, rdb)

	// Initialize Kafka Consumer
	consumer := kafka.NewConsumer([]string{cfg.KafkaBrokers}, "fraud_group", "payment_events")

	// Start consuming in a goroutine
	ctx, cancelConsume := context.WithCancel(context.Background())
	defer cancelConsume()

	go func() {
		log.Println("Fraud service started. Listening for payment events...")
		consumer.Consume(ctx, func(msg segmentioKafka.Message) error {
			return fraudSvc.HandlePaymentEvent(msg.Value)
		})
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Fraud service...")
	cancelConsume()
	consumer.Close()
	log.Println("Fraud service exited cleanly")
}
