package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"payment-processor/internal/database"
	"payment-processor/internal/kafka"
	"payment-processor/pkg/uid"
)

type SettlementService struct {
	db       *database.DB
	producer *kafka.Producer
}

func NewSettlementService(db *database.DB, producer *kafka.Producer) *SettlementService {
	return &SettlementService{
		db:       db,
		producer: producer,
	}
}

// StartScheduler polls for initiated transactions that are due for settlement
func (s *SettlementService) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollAndDispatchSettlements(ctx)
		}
	}
}

func (s *SettlementService) pollAndDispatchSettlements(ctx context.Context) {
	// Query: Find payments in 'initiated' state that should be settled now.
	// Rule: updated_at + merchant.payout_schedule <= NOW()
	// Since payout_schedule is an enum, we use a CASE statement to add the appropriate interval.
	query := `
		SELECT p.merchant_id, p.currency, SUM(p.amount) as total_amount, array_agg(p.id) as payment_ids
		FROM payments p
		JOIN merchants m ON p.merchant_id = m.id
		WHERE p.status = 'initiated' AND p.settlement_id IS NULL
		AND (
			(m.payout_schedule = 'instant') OR
			(m.payout_schedule = '1_hour' AND p.updated_at + INTERVAL '1 hour' <= NOW()) OR
			(m.payout_schedule = '12_hours' AND p.updated_at + INTERVAL '12 hours' <= NOW()) OR
			(m.payout_schedule = '24_hours' AND p.updated_at + INTERVAL '24 hours' <= NOW()) OR
			(m.payout_schedule IN ('daily', 'weekly', 'biweekly', 'monthly') AND p.updated_at + INTERVAL '2 days' <= NOW()) -- fallback for legacy
		)
		GROUP BY p.merchant_id, p.currency
		LIMIT 100
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("Error polling for settlements: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mID, currency string
		var amount int64
		var paymentIDs []string
		if err := rows.Scan(&mID, &currency, &amount, &paymentIDs); err != nil {
			log.Printf("Error scanning settlement row: %v", err)
			continue
		}

		// We skip updating the DB here to avoid FK constraints on 'pending'
		// The Kafka worker will check settlement_id IS NULL to ensure idempotency.

		event := map[string]interface{}{
			"payment_ids": paymentIDs,
			"merchant_id": mID,
			"amount":      amount,
			"currency":    currency,
			"status":      "initiated",
		}

		// Use merchantID as key to ensure ordering per merchant
		err = s.producer.Publish(ctx, mID, event)
		if err != nil {
			log.Printf("Error publishing settlement event for merchant %s: %v", mID, err)
		} else {
			log.Printf("Dispatched bulk settlement for merchant %s with %d payments", mID, len(paymentIDs))
		}
	}
}

type SettleEvent struct {
	PaymentID  string   `json:"payment_id,omitempty"` // For backwards compatibility
	PaymentIDs []string `json:"payment_ids,omitempty"`
	MerchantID string   `json:"merchant_id"`
	Amount     int64    `json:"amount"`
	Currency   string   `json:"currency"`
	Status     string   `json:"status"`
}

// HandleSettleEvent consumes 'settling' events from Kafka and performs the final settlement
func (s *SettlementService) HandleSettleEvent(msg kafkago.Message) error {
	var event SettleEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal settle event: %w", err)
	}

	if event.Status != "initiated" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Create Settlement Record
	settlementID := uid.Generate("set_")
	_, err = tx.Exec(ctx, `
		INSERT INTO settlements (id, merchant_id, amount, currency, status) 
		VALUES ($1, $2, $3, $4, 'processed')
	`, settlementID, event.MerchantID, event.Amount, event.Currency)
	if err != nil {
		return fmt.Errorf("failed to create settlement record: %w", err)
	}

	// Get all payment IDs to process
	var targetPaymentIDs []string
	if len(event.PaymentIDs) > 0 {
		targetPaymentIDs = event.PaymentIDs
	} else if event.PaymentID != "" {
		targetPaymentIDs = []string{event.PaymentID}
	}

	if len(targetPaymentIDs) == 0 {
		return nil
	}

	// Update Payments with the generated SettlementID and mark them as processed
	res, err := tx.Exec(ctx, "UPDATE payments SET settlement_id = $1, status = 'processed', updated_at = NOW() WHERE id = ANY($2) AND settlement_id IS NULL", settlementID, targetPaymentIDs)
	if err != nil {
		return fmt.Errorf("failed to attach settlement id: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil // Already settled
	}

	// 3. Double-Entry Bookkeeping
	// Debit: Merchant Payable (We owed them)
	// Credit: Merchant Settlement (We paid them into their actual bank)

	var payableAccID, settlementAccID string

	// Ensure Payable Account exists
	err = tx.QueryRow(ctx, "SELECT id FROM accounts WHERE merchant_id = $1 AND type = 'merchant_payable' AND currency = $2", event.MerchantID, event.Currency).Scan(&payableAccID)
	if err != nil {
		// Create it if it doesn't exist
		payableAccID = uid.Generate("acc_")
		_, err = tx.Exec(ctx, "INSERT INTO accounts (id, merchant_id, type, currency) VALUES ($1, $2, 'merchant_payable', $3)", payableAccID, event.MerchantID, event.Currency)
		if err != nil {
			return fmt.Errorf("failed to create merchant payable account: %w", err)
		}
	}

	// Ensure Settlement Account exists
	err = tx.QueryRow(ctx, "SELECT id FROM accounts WHERE merchant_id = $1 AND type = 'merchant_settlement' AND currency = $2", event.MerchantID, event.Currency).Scan(&settlementAccID)
	if err != nil {
		settlementAccID = uid.Generate("acc_")
		_, err = tx.Exec(ctx, "INSERT INTO accounts (id, merchant_id, type, currency) VALUES ($1, $2, 'merchant_settlement', $3)", settlementAccID, event.MerchantID, event.Currency)
		if err != nil {
			return fmt.Errorf("failed to create settlement account: %w", err)
		}
	}

	transactionGroupID := uid.Generate("txg_")

	// Debit Payable
	debitEntryID := uid.Generate("le_")
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, transaction_group_id, account_id, entry_type, amount, currency)
		VALUES ($1, $2, $3, 'debit', $4, $5)
	`, debitEntryID, transactionGroupID, payableAccID, event.Amount, event.Currency)
	if err != nil {
		return fmt.Errorf("failed to create debit ledger entry: %w", err)
	}

	// Credit Settlement
	creditEntryID := uid.Generate("le_")
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, transaction_group_id, account_id, entry_type, amount, currency)
		VALUES ($1, $2, $3, 'credit', $4, $5)
	`, creditEntryID, transactionGroupID, settlementAccID, event.Amount, event.Currency)
	if err != nil {
		return fmt.Errorf("failed to create credit ledger entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit settlement transaction: %w", err)
	}
	log.Printf("Settlement %s completed successfully for %d payments", settlementID, len(targetPaymentIDs))

	for _, pID := range targetPaymentIDs {
		webhookEvent := map[string]interface{}{
			"event_type": "payment.settled",
			"payment_id": pID,
			"status":     "settled",
			"amount":     event.Amount, // We might need to split this but for now it's okay
			"currency":   event.Currency,
		}

		go func(pid string) {
			_ = s.producer.Publish(context.Background(), pid, webhookEvent)
		}(pID)
	}

	return nil
}
