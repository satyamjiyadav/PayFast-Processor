package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"payment-processor/internal/database"
)

type WebhookService struct {
	db     *database.DB
	client *http.Client
}

func NewWebhookService(db *database.DB) *WebhookService {
	return &WebhookService{
		db: db,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type WebhookEvent struct {
	EventType string `json:"event_type"`
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

func (s *WebhookService) HandleWebhookEvent(eventData []byte) error {
	var event WebhookEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal webhook event: %w", err)
	}

	log.Printf("Received webhook event: %s for payment %s", event.EventType, event.PaymentID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Fetch Merchant's Webhook URL and Secret
	var merchantID string
	err := s.db.Pool.QueryRow(ctx, "SELECT merchant_id FROM payments WHERE id = $1", event.PaymentID).Scan(&merchantID)
	if err != nil {
		return fmt.Errorf("failed to find merchant for payment %s: %w", event.PaymentID, err)
	}

	var webhookURL, webhookSecret sql.NullString
	err = s.db.Pool.QueryRow(ctx, "SELECT webhook_url, webhook_secret_hash FROM merchants WHERE id = $1", merchantID).Scan(&webhookURL, &webhookSecret)
	if err != nil {
		return fmt.Errorf("failed to find webhook info for merchant %s: %w", merchantID, err)
	}

	if !webhookURL.Valid || webhookURL.String == "" {
		log.Printf("No webhook URL configured for merchant %s, skipping", merchantID)
		return nil // Not an error if they haven't configured it
	}

	// 2. Prepare Payload
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 3. Sign Payload (HMAC SHA256)
	signature := signPayload(payload, webhookSecret.String)

	// 4. Send HTTP Request with Exponential Backoff Retries
	maxRetries := 3
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		success, err := s.sendWebhook(ctx, webhookURL.String, payload, signature)
		if success {
			log.Printf("Successfully delivered webhook for payment %s to %s", event.PaymentID, webhookURL.String)
			return nil
		}
		
		log.Printf("Attempt %d failed to deliver webhook: %v", attempt, err)
		if attempt < maxRetries {
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("failed to deliver webhook after %d attempts", maxRetries)
}

func (s *WebhookService) sendWebhook(ctx context.Context, url string, payload []byte, signature string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Signature", signature)
	req.Header.Set("User-Agent", "PaymentProcessor-Webhook/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	return false, fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
}

func signPayload(payload []byte, secret string) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
