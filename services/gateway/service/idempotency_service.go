package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"payment-processor/internal/database"
	"payment-processor/internal/redis"
)

type IdempotencyService struct {
	db    *database.DB
	redis *redis.Client
}

func NewIdempotencyService(db *database.DB, redis *redis.Client) *IdempotencyService {
	return &IdempotencyService{
		db:    db,
		redis: redis,
	}
}

// AcquireLock attempts to acquire a distributed lock for a specific key.
func (s *IdempotencyService) AcquireLock(ctx context.Context, merchantID, key string) (bool, error) {
	lockKey := fmt.Sprintf("idempotency_lock:%s:%s", merchantID, key)
	// Lock for 30 seconds
	return s.redis.SetNX(ctx, lockKey, "locked", 30*time.Second)
}

// ReleaseLock releases the distributed lock.
func (s *IdempotencyService) ReleaseLock(ctx context.Context, merchantID, key string) error {
	lockKey := fmt.Sprintf("idempotency_lock:%s:%s", merchantID, key)
	return s.redis.Del(ctx, lockKey)
}

type IdempotencyRecord struct {
	Key          string
	Status       string
	ResponseCode int
	ResponseBody []byte
}

// GetRecord fetches the idempotency record from DB.
func (s *IdempotencyService) GetRecord(ctx context.Context, merchantID, key string) (*IdempotencyRecord, error) {
	query := `SELECT status, response_code, response_body FROM idempotency_keys WHERE merchant_id = $1 AND key = $2`
	row := s.db.Pool.QueryRow(ctx, query, merchantID, key)

	var record IdempotencyRecord
	var respBody []byte
	var respCode *int16

	err := row.Scan(&record.Status, &respCode, &respBody)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Not found
		}
		return nil, err
	}

	record.Key = key
	if respCode != nil {
		record.ResponseCode = int(*respCode)
	}
	record.ResponseBody = respBody

	return &record, nil
}

// CreatePendingRecord inserts a new idempotency key with 'in_progress' status.
func (s *IdempotencyService) CreatePendingRecord(ctx context.Context, merchantID, key, path, method string) error {
	query := `
		INSERT INTO idempotency_keys (key, merchant_id, request_path, request_method, status)
		VALUES ($1, $2, $3, $4, 'in_progress')
	`
	_, err := s.db.Pool.Exec(ctx, query, key, merchantID, path, method)
	return err
}

// CompleteRecord updates the idempotency key with the final response.
func (s *IdempotencyService) CompleteRecord(ctx context.Context, merchantID, key string, statusCode int, responseBody interface{}) error {
	bodyBytes, err := json.Marshal(responseBody)
	if err != nil {
		return err
	}

	query := `
		UPDATE idempotency_keys
		SET status = 'completed', response_code = $1, response_body = $2
		WHERE merchant_id = $3 AND key = $4
	`
	_, err = s.db.Pool.Exec(ctx, query, statusCode, bodyBytes, merchantID, key)
	return err
}
