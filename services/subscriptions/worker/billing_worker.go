package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"payment-processor/internal/database"
	"payment-processor/pkg/uid"
)

type BillingWorker struct {
	db         *database.DB
	gatewayURL string // e.g. http://localhost/v1/payments
}

func NewBillingWorker(db *database.DB, gatewayURL string) *BillingWorker {
	return &BillingWorker{
		db:         db,
		gatewayURL: gatewayURL,
	}
}

func (w *BillingWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Println("Billing worker started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Billing worker stopped")
			return
		case <-ticker.C:
			w.processDueSubscriptions(ctx)
		}
	}
}

func (w *BillingWorker) processDueSubscriptions(ctx context.Context) {
	// Find all active subscriptions where current_period_end is in the past
	query := `
		SELECT id, merchant_id, payment_method_id, amount, currency, interval 
		FROM subscriptions 
		WHERE status = 'active' AND current_period_end <= NOW()
		LIMIT 100
	`
	rows, err := w.db.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to query due subscriptions: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var subID, merchantID, paymentMethodID, currency, interval string
		var amount int64
		if err := rows.Scan(&subID, &merchantID, &paymentMethodID, &amount, &currency, &interval); err != nil {
			log.Printf("Failed to scan subscription: %v", err)
			continue
		}

		w.chargeSubscription(ctx, subID, merchantID, paymentMethodID, amount, currency, interval)
	}
}

func (w *BillingWorker) chargeSubscription(ctx context.Context, subID, merchantID, paymentMethodID string, amount int64, currency, interval string) {
	log.Printf("Charging subscription %s for %d %s", subID, amount, currency)

	// Create a draft invoice
	invoiceID := uid.Generate("in_")
	_, err := w.db.Pool.Exec(ctx, `
		INSERT INTO invoices (id, subscription_id, merchant_id, amount, currency, status)
		VALUES ($1, $2, $3, $4, $5, 'draft')
	`, invoiceID, subID, merchantID, amount, currency)
	
	if err != nil {
		log.Printf("Failed to create invoice for sub %s: %v", subID, err)
		return
	}

	// Call Gateway API to process payment
	reqBody, _ := json.Marshal(map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"token_id": paymentMethodID,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.gatewayURL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("Failed to create request for sub %s: %v", subID, err)
		return
	}

	// In a real system we would look up the merchant's API key
	req.Header.Set("Authorization", "Bearer sk_test_123") 
	req.Header.Set("Content-Type", "application/json")
	// Idempotency key prevents double charging the same invoice
	req.Header.Set("Idempotency-Key", "idem_invoice_"+invoiceID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	
	if err != nil || resp.StatusCode >= 400 {
		log.Printf("Payment failed for invoice %s (sub %s): %v", invoiceID, subID, err)
		w.db.Pool.Exec(ctx, "UPDATE invoices SET status = 'uncollectible' WHERE id = $1", invoiceID)
		w.db.Pool.Exec(ctx, "UPDATE subscriptions SET status = 'past_due' WHERE id = $1", subID)
		return
	}
	defer resp.Body.Close()

	var paymentResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&paymentResp)
	paymentID, _ := paymentResp["payment_id"].(string)

	// Update Invoice and Subscription
	tx, _ := w.db.Pool.Begin(ctx)
	defer tx.Rollback(ctx)

	tx.Exec(ctx, "UPDATE invoices SET status = 'paid', payment_id = $1 WHERE id = $2", paymentID, invoiceID)
	
	// Extend subscription period
	extendQuery := `
		UPDATE subscriptions 
		SET current_period_start = current_period_end,
			current_period_end = current_period_end + (CASE WHEN interval = 'month' THEN interval '1 month' ELSE interval '1 year' END)
		WHERE id = $1
	`
	tx.Exec(ctx, extendQuery, subID)
	tx.Commit(ctx)

	log.Printf("Successfully charged subscription %s, invoice %s", subID, invoiceID)
}
