package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, groupID string, topic string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		Topic:       topic,
		StartOffset: kafka.LastOffset, // Only process new messages
	})

	return &Consumer{reader: r}
}

// Consume starts listening for messages and calls the handler function
func (c *Consumer) Consume(ctx context.Context, handler func(msg kafka.Message) error) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer stopped by context")
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return // Context cancelled during fetch
				}
				log.Printf("Failed to fetch message: %v", err)
				continue
			}

			if err := handler(msg); err != nil {
				log.Printf("Error processing message key %s: %v", string(msg.Key), err)
				// Note: In production, consider DLQ logic here
			} else {
				// Commit only if successful
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Failed to commit message: %v", err)
				}
			}
		}
	}
}

func (c *Consumer) Close() {
	if c.reader != nil {
		c.reader.Close()
	}
}
