package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"payment-processor/internal/database"
)

type AdminService struct {
	db *database.DB
}

func NewAdminService(db *database.DB) *AdminService {
	return &AdminService{db: db}
}

type PaymentRecord struct {
	ID           string    `json:"id"`
	Amount       int64     `json:"amount"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	SettlementID *string   `json:"settlement_id"`
}

type SettlementRecord struct {
	ID         string    `json:"id"`
	MerchantID string    `json:"merchant_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type LedgerRecord struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	PaymentID string    `json:"payment_id"`
	EntryType string    `json:"entry_type"` // debit or credit
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type SubscriptionRecord struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"`
	Interval         string    `json:"interval"`
	Amount           int64     `json:"amount"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
}

func (s *AdminService) GetPayments(ctx context.Context) ([]PaymentRecord, error) {
	rows, err := s.db.Pool.Query(ctx, "SELECT id, amount, currency, status, created_at, settlement_id FROM payments ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		return nil, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	var payments []PaymentRecord
	for rows.Next() {
		var p PaymentRecord
		if err := rows.Scan(&p.ID, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt, &p.SettlementID); err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (s *AdminService) GetSettlements(ctx context.Context) ([]SettlementRecord, error) {
	rows, err := s.db.Pool.Query(ctx, "SELECT id, merchant_id, amount, currency, status, created_at FROM settlements ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		return nil, fmt.Errorf("failed to query settlements: %w", err)
	}
	defer rows.Close()

	var settlements []SettlementRecord
	for rows.Next() {
		var sRec SettlementRecord
		if err := rows.Scan(&sRec.ID, &sRec.MerchantID, &sRec.Amount, &sRec.Currency, &sRec.Status, &sRec.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan settlement: %w", err)
		}
		settlements = append(settlements, sRec)
	}
	return settlements, nil
}

func (s *AdminService) GetRecentLedgerEntries(ctx context.Context) ([]LedgerRecord, error) {
	query := `SELECT id, account_id, payment_id, entry_type, amount, created_at FROM ledger_entries ORDER BY created_at DESC LIMIT 50`
	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []LedgerRecord
	for rows.Next() {
		var e LedgerRecord
		if err := rows.Scan(&e.ID, &e.AccountID, &e.PaymentID, &e.EntryType, &e.Amount, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *AdminService) GetSubscriptions(ctx context.Context) ([]SubscriptionRecord, error) {
	query := `SELECT id, status, interval, amount, current_period_end FROM subscriptions ORDER BY created_at DESC LIMIT 20`
	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []SubscriptionRecord
	for rows.Next() {
		var sub SubscriptionRecord
		if err := rows.Scan(&sub.ID, &sub.Status, &sub.Interval, &sub.Amount, &sub.CurrentPeriodEnd); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

type MerchantSettings struct {
	PayoutSchedule string `json:"payout_schedule"`
}

func (s *AdminService) GetMerchantSettings(ctx context.Context, merchantID string) (*MerchantSettings, error) {
	var schedule string
	err := s.db.Pool.QueryRow(ctx, "SELECT payout_schedule FROM merchants WHERE id = $1", merchantID).Scan(&schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to query merchant settings: %w", err)
	}
	return &MerchantSettings{PayoutSchedule: schedule}, nil
}

func (s *AdminService) UpdateMerchantSettings(ctx context.Context, merchantID, payoutSchedule string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE merchants 
		SET payout_schedule = $1, updated_at = NOW() 
		WHERE id = $2`, payoutSchedule, merchantID)

	if err == nil && payoutSchedule == "instant" {
		// Trigger immediate settlement for any pending initiated payments
		_ = s.settlePendingPayments(ctx, merchantID)
	}
	return err
}

func (s *AdminService) settlePendingPayments(ctx context.Context, merchantID string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Sum total amount of pending initiated payments
	var totalAmount int64
	var count int
	err = tx.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0), COUNT(id) FROM payments WHERE merchant_id = $1 AND status = 'initiated' AND settlement_id IS NULL", merchantID).Scan(&totalAmount, &count)
	if err != nil || count == 0 {
		return nil // No pending payments
	}

	// Create Settlement Record
	settlementID := "set_" + fmt.Sprintf("%d", time.Now().UnixNano())
	_, err = tx.Exec(ctx, `
		INSERT INTO settlements (id, merchant_id, amount, currency, status) 
		VALUES ($1, $2, $3, 'USD', 'processed')
	`, settlementID, merchantID, totalAmount)
	if err != nil {
		return fmt.Errorf("failed to create settlement: %w", err)
	}

	// Attach settlement ID to payments
	_, err = tx.Exec(ctx, "UPDATE payments SET settlement_id = $1, status = 'processed', updated_at = NOW() WHERE merchant_id = $2 AND status = 'initiated' AND settlement_id IS NULL", settlementID, merchantID)
	if err != nil {
		return fmt.Errorf("failed to attach settlement id: %w", err)
	}

	// Create Ledger Entry for Double Entry Bookkeeping
	transactionGroupID := "txg_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Debit Merchant Payable
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, account_id, payment_id, transaction_group_id, entry_type, amount)
		VALUES ($1, $2, $3, $4, 'debit', $5)
	`, "led_"+fmt.Sprintf("%d", time.Now().UnixNano())+"_1", "acc_merchant_payable_"+merchantID, nil, transactionGroupID, totalAmount)
	if err != nil {
		return fmt.Errorf("failed ledger debit: %w", err)
	}

	// Credit Merchant Settlement Bank Account
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, account_id, payment_id, transaction_group_id, entry_type, amount)
		VALUES ($1, $2, $3, $4, 'credit', $5)
	`, "led_"+fmt.Sprintf("%d", time.Now().UnixNano())+"_2", "acc_merchant_settlement_"+merchantID, nil, transactionGroupID, totalAmount)
	if err != nil {
		return fmt.Errorf("failed ledger credit: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *AdminService) GenerateSettlementsReport(ctx context.Context, merchantID string, timeRange string) (string, error) {
	var interval string
	switch timeRange {
	case "24h":
		interval = "24 hours"
	case "48h":
		interval = "48 hours"
	case "7d":
		interval = "7 days"
	case "30d":
		interval = "30 days"
	default:
		interval = "24 hours"
	}

	query := `
		SELECT id, amount, currency, status, created_at, updated_at
		FROM settlements
		WHERE merchant_id = $1 AND created_at >= NOW() - INTERVAL '` + interval + `'
		ORDER BY created_at DESC
	`
	rows, err := s.db.Pool.Query(ctx, query, merchantID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch settlements: %w", err)
	}
	defer rows.Close()

	var csvData string
	csvData += "Settlement ID,Amount,Currency,Status,Created At,Updated At\n"

	for rows.Next() {
		var id, currency, status string
		var amount int64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &amount, &currency, &status, &createdAt, &updatedAt); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		
		amountFormatted := fmt.Sprintf("%.2f", float64(amount)/100.0)
		csvData += fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			id, amountFormatted, currency, status, createdAt.Format(time.RFC3339), updatedAt.Format(time.RFC3339))
	}

	return csvData, nil
}

type Refund struct {
	ID         string    `json:"id"`
	PaymentID  string    `json:"payment_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *AdminService) GetRefunds(ctx context.Context, merchantID string) ([]Refund, error) {
	rows, err := s.db.Pool.Query(ctx, "SELECT id, payment_id, amount, currency, status, created_at FROM refunds WHERE merchant_id = $1 ORDER BY created_at DESC", merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch refunds: %w", err)
	}
	defer rows.Close()

	var refunds []Refund
	for rows.Next() {
		var r Refund
		if err := rows.Scan(&r.ID, &r.PaymentID, &r.Amount, &r.Currency, &r.Status, &r.CreatedAt); err != nil {
			continue
		}
		refunds = append(refunds, r)
	}
	return refunds, nil
}

func (s *AdminService) IssueRefund(ctx context.Context, merchantID, paymentID string, amount int64) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check payment status
	var currentStatus string
	var currentAmount int64
	err = tx.QueryRow(ctx, "SELECT status, amount FROM payments WHERE id = $1 AND merchant_id = $2", paymentID, merchantID).Scan(&currentStatus, &currentAmount)
	if err != nil {
		return fmt.Errorf("payment not found")
	}
	if currentStatus != "processed" {
		return fmt.Errorf("can only refund processed payments")
	}

	// For simplicity, we just do full refunds here, so we verify amount
	if amount != currentAmount {
		return fmt.Errorf("partial refunds not yet supported via admin UI (requested %d, original %d)", amount, currentAmount)
	}

	refundID := fmt.Sprintf("ref_%d", time.Now().UnixNano())

	// Create refund record
	_, err = tx.Exec(ctx, "INSERT INTO refunds (id, payment_id, merchant_id, amount, currency, status) VALUES ($1, $2, $3, $4, 'usd', 'processed')", refundID, paymentID, merchantID, amount)
	if err != nil {
		return fmt.Errorf("failed to insert refund: %w", err)
	}

	// Update payment status
	_, err = tx.Exec(ctx, "UPDATE payments SET status = 'refunded' WHERE id = $1", paymentID)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	txGroupID := "tx_ref_" + refundID

	// Get merchant account ID
	var merchantAccountID string
	err = tx.QueryRow(ctx, "SELECT id FROM accounts WHERE merchant_id = $1 AND type = 'merchant_payable' AND currency = 'usd'", merchantID).Scan(&merchantAccountID)
	if err != nil {
		return fmt.Errorf("failed to find merchant account: %w", err)
	}

	systemAccountID := "acc_system_01"

	// Debit merchant settlement account (reducing their balance because they are returning money)
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, account_id, amount, currency, transaction_group_id, entry_type)
		VALUES ($1, $2, $3, 'usd', $4, 'debit')
	`, "led_"+fmt.Sprintf("%d", time.Now().UnixNano()), merchantAccountID, amount, txGroupID)
	
	if err != nil {
		return fmt.Errorf("failed ledger debit: %w", err)
	}

	// Credit outbound network account (returning money to user)
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, account_id, amount, currency, transaction_group_id, entry_type)
		VALUES ($1, $2, $3, 'usd', $4, 'credit')
	`, "led_"+fmt.Sprintf("%d", time.Now().UnixNano())+"_2", systemAccountID, amount, txGroupID)
	
	if err != nil {
		return fmt.Errorf("failed ledger credit: %w", err)
	}

	return tx.Commit(ctx)
}
