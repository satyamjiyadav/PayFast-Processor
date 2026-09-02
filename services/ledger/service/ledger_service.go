package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"payment-processor/internal/database"
	"payment-processor/internal/kafka"
	"payment-processor/pkg/uid"
)

type LedgerService struct {
	db       *database.DB
	producer *kafka.Producer
}

func NewLedgerService(db *database.DB, producer *kafka.Producer) *LedgerService {
	return &LedgerService{
		db:       db,
		producer: producer,
	}
}

type PaymentEvent struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	TokenID   string `json:"token_id"`
	Status    string `json:"status"`
}

// HandlePaymentEvent consumes Kafka messages and performs double-entry bookkeeping
func (s *LedgerService) HandlePaymentEvent(eventData []byte) error {
	var event PaymentEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if event.Status != "created" {
		log.Printf("Ledger ignoring event with status: %s", event.Status)
		return nil
	}

	log.Printf("Processing double-entry ledger for processed payment %s amount %d %s", event.PaymentID, event.Amount, event.Currency)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// We no longer update the payment status here, because Gateway already did that.
	// We only record the double entry bookkeeping.

	// 1. Get Merchant ID for the payment
	var merchantID string
	err = tx.QueryRow(ctx, "SELECT merchant_id FROM payments WHERE id = $1", event.PaymentID).Scan(&merchantID)
	if err != nil {
		return fmt.Errorf("failed to get merchant_id: %w", err)
	}

	// Set payment status to 'initiated' after ledger entry is created
	_, err = tx.Exec(ctx, "UPDATE payments SET status = 'initiated', updated_at = NOW() WHERE id = $1", event.PaymentID)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// 2. Ensure Merchant Payable Account Exists
	var merchantAccountID string
	err = tx.QueryRow(ctx, "SELECT id FROM accounts WHERE merchant_id = $1 AND type = 'merchant_payable' AND currency = $2", merchantID, event.Currency).Scan(&merchantAccountID)
		if err != nil {
			// Create it if it doesn't exist
			merchantAccountID = uid.Generate("acc_")
			_, err = tx.Exec(ctx, "INSERT INTO accounts (id, merchant_id, type, currency) VALUES ($1, $2, 'merchant_payable', $3)", merchantAccountID, merchantID, event.Currency)
			if err != nil {
				return fmt.Errorf("failed to create merchant account: %w", err)
			}
		}

		// 3. Double-Entry Bookkeeping:
		// Debit: Network/Acquiring Bank Holding Account (System Account)
		// Credit: Merchant Payable Account

		transactionGroupID := uid.Generate("txg_")

		systemAccountID := "acc_system_01"
		// Ensure system account exists
		var sysAccExists bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)", systemAccountID).Scan(&sysAccExists)
		if err != nil {
			return fmt.Errorf("failed to check system account: %w", err)
		}
		if !sysAccExists {
			_, err = tx.Exec(ctx, "INSERT INTO accounts (id, type, currency) VALUES ($1, 'bank_transit', $2)", systemAccountID, event.Currency)
			if err != nil {
				return fmt.Errorf("failed to create system account: %w", err)
			}
		}
		
		debitEntryID := uid.Generate("le_")
		_, err = tx.Exec(ctx, `
			INSERT INTO ledger_entries (id, transaction_group_id, account_id, payment_id, entry_type, amount, currency)
			VALUES ($1, $2, $3, $4, 'debit', $5, $6)
		`, debitEntryID, transactionGroupID, systemAccountID, event.PaymentID, event.Amount, event.Currency)
		if err != nil {
			return fmt.Errorf("failed to create debit ledger entry: %w", err)
		}

		// Create Credit Entry
		creditEntryID := uid.Generate("le_")
		_, err = tx.Exec(ctx, `
			INSERT INTO ledger_entries (id, transaction_group_id, account_id, payment_id, entry_type, amount, currency)
			VALUES ($1, $2, $3, $4, 'credit', $5, $6)
		`, creditEntryID, transactionGroupID, merchantAccountID, event.PaymentID, event.Amount, event.Currency)
		if err != nil {
			return fmt.Errorf("failed to create credit ledger entry: %w", err)
		}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit ledger transaction: %w", err)
	}

	log.Printf("Ledger entries created for Payment %s", event.PaymentID)

	// Publish to webhook_events
	webhookEvent := map[string]interface{}{
		"event_type": "payment.initiated",
		"payment_id": event.PaymentID,
		"status":     "initiated",
		"amount":     event.Amount,
		"currency":   event.Currency,
	}

	go func() {
		err := s.producer.Publish(context.Background(), event.PaymentID, webhookEvent)
		if err != nil {
			log.Printf("Failed to publish webhook event: %v", err)
		}
	}()

	return nil
}
