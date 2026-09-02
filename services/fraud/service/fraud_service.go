package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"payment-processor/internal/database"
	"payment-processor/internal/redis"
)

type FraudService struct {
	db  *database.DB
	rdb *redis.Client
}

func NewFraudService(db *database.DB, rdb *redis.Client) *FraudService {
	return &FraudService{
		db:  db,
		rdb: rdb,
	}
}

type PaymentEvent struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	TokenID   string `json:"token_id"`
}

// HandlePaymentEvent consumes Kafka messages and performs fraud analysis
func (s *FraudService) HandlePaymentEvent(eventData []byte) error {
	var event PaymentEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Fraud check only for created or authorized payments
	if event.Status != "created" && event.Status != "authorized" {
		return nil
	}

	log.Printf("Analyzing fraud risk for payment %s (token: %s)...", event.PaymentID, event.TokenID)

	// Velocity Check via Redis
	var riskScore int
	velocityKey := fmt.Sprintf("fraud_velocity:token:%s", event.TokenID)
	
	// Increment the attempt count for this token
	attempts, err := s.rdb.Client.Incr(context.Background(), velocityKey).Result()
	if err == nil && attempts == 1 {
		// Set a 60 second expiration window on the first attempt
		s.rdb.Client.Expire(context.Background(), velocityKey, 60*time.Second)
	}

	if attempts > 3 {
		log.Printf("Velocity limit exceeded for token %s. Marking high risk.", event.TokenID)
		riskScore = 90
	} else {
		// Simulate Fraud Detection Algorithm (Mock)
		riskScore = simulateFraudCheck(event.Amount)
	}

	// Update the payment record asynchronously
	query := `UPDATE payments SET risk_score = $1 WHERE id = $2`
	_, err = s.db.Pool.Exec(context.Background(), query, riskScore, event.PaymentID)
	if err != nil {
		return fmt.Errorf("failed to update risk score: %w", err)
	}

	log.Printf("Risk score %d assigned to payment %s", riskScore, event.PaymentID)

	if riskScore > 80 {
		log.Printf("ALERT: High risk payment detected! ID: %s, Score: %d", event.PaymentID, riskScore)
		// In production, publish to an alerts topic or notify risk analysts
	}

	return nil
}

func simulateFraudCheck(amount int64) int {
	// Simple heuristic: higher amounts have slightly higher chance of high risk score
	baseScore := rand.Intn(50)
	if amount > 100000 { // $1000.00
		baseScore += 30
	}
	return baseScore
}
