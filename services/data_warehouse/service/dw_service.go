package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type DataWarehouseService struct {
	dataDir string
}

func NewDataWarehouseService(dataDir string) *DataWarehouseService {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data warehouse directory: %v", err)
	}
	return &DataWarehouseService{
		dataDir: dataDir,
	}
}

type PaymentEvent struct {
	PaymentID  string `json:"payment_id"`
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Status     string `json:"status"`
}

// HandlePaymentEvent consumes Kafka messages and simulates a data warehouse ETL process
func (s *DataWarehouseService) HandlePaymentEvent(eventData []byte) error {
	var event PaymentEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("Archiving payment %s to Data Warehouse...", event.PaymentID)

	// In a real system, this might write to an S3 bucket (Data Lake), BigQuery, or Snowflake.
	// For this simulation, we'll append to a daily JSON Lines file.
	dateStr := time.Now().Format("2006-01-02")
	fileName := fmt.Sprintf("payments_archive_%s.jsonl", dateStr)
	filePath := filepath.Join(s.dataDir, fileName)

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer file.Close()

	archiveRecord := map[string]interface{}{
		"event_time": time.Now().Format(time.RFC3339),
		"payment":    event,
	}

	recordBytes, err := json.Marshal(archiveRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal archive record: %w", err)
	}

	if _, err := file.Write(append(recordBytes, '\n')); err != nil {
		return fmt.Errorf("failed to write to archive file: %w", err)
	}

	return nil
}
