package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"payment-processor/internal/config"
	"payment-processor/internal/kafka"
	"payment-processor/services/data_warehouse/service"
	segmentioKafka "github.com/segmentio/kafka-go"
)

func main() {
	cfg := config.Load()

	// Initialize Services
	dwDir := "./data/warehouse"
	dwSvc := service.NewDataWarehouseService(dwDir)

	// Initialize Kafka Consumer
	consumer := kafka.NewConsumer([]string{cfg.KafkaBrokers}, "dw_group", "payment_events")

	// Start consuming in a goroutine
	ctx, cancelConsume := context.WithCancel(context.Background())
	defer cancelConsume()

	go func() {
		log.Println("Data Warehouse service started. Listening for payment events...")
		consumer.Consume(ctx, func(msg segmentioKafka.Message) error {
			return dwSvc.HandlePaymentEvent(msg.Value)
		})
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Data Warehouse service...")
	cancelConsume()
	consumer.Close()
	log.Println("Data Warehouse service exited cleanly")
}
