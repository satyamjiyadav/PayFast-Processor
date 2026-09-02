package service

import (
	"context"
	"fmt"
	"time"

	"payment-processor/internal/database"
	"payment-processor/pkg/uid"
)

type SubscriptionService struct {
	db *database.DB
}

func NewSubscriptionService(db *database.DB) *SubscriptionService {
	return &SubscriptionService{db: db}
}

type CreateSubscriptionRequest struct {
	MerchantID      string `json:"merchant_id"`
	CustomerID      string `json:"customer_id"`
	PaymentMethodID string `json:"payment_method_id"`
	PlanID          string `json:"plan_id"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	Interval        string `json:"interval"` // 'month' or 'year'
}

type Subscription struct {
	ID                 string    `json:"id"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*Subscription, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if req.Interval != "month" && req.Interval != "year" {
		return nil, fmt.Errorf("interval must be 'month' or 'year'")
	}

	subID := uid.Generate("sub_")
	now := time.Now()
	var end time.Time
	if req.Interval == "month" {
		end = now.AddDate(0, 1, 0)
	} else {
		end = now.AddDate(1, 0, 0)
	}

	query := `
		INSERT INTO subscriptions (id, merchant_id, customer_id, payment_method_id, plan_id, amount, currency, interval, status, current_period_start, current_period_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', $9, $10)
	`
	_, err := s.db.Pool.Exec(ctx, query,
		subID, req.MerchantID, req.CustomerID, req.PaymentMethodID, req.PlanID, req.Amount, req.Currency, req.Interval, now, end,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return &Subscription{
		ID:                 subID,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   end,
	}, nil
}
