package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"payment-processor/internal/database"
	"payment-processor/pkg/crypto"
	"payment-processor/pkg/uid"
)

type VaultService struct {
	db       *database.DB
	vaultKey []byte // 32-byte AES key
}

func NewVaultService(db *database.DB, vaultKey string) *VaultService {
	return &VaultService{
		db:       db,
		vaultKey: []byte(vaultKey),
	}
}

type TokenizeRequest struct {
	MerchantID string `json:"merchant_id"`
	Type       string `json:"type"` // "card", "upi", "netbanking"
	PAN        string `json:"pan"` // Personal Account Number (16 digits)
	ExpMonth   int    `json:"exp_month"`
	ExpYear    int    `json:"exp_year"`
	UpiVpa     string `json:"upi_vpa"` // e.g. user@upi
	BankCode   string `json:"bank_code"` // e.g. HDFC, SBI
}

type TokenizeResponse struct {
	TokenID  string `json:"token_id"` // pm_xxx
	LastFour string `json:"last_four"`
}

func (s *VaultService) TokenizeCard(ctx context.Context, req TokenizeRequest) (*TokenizeResponse, error) {
	if req.Type == "" {
		req.Type = "card" // Default for backwards compatibility
	}

	tokenID := uid.Generate("pm_")
	
	if req.Type == "card" {
		if len(req.PAN) < 13 || len(req.PAN) > 19 {
			return nil, fmt.Errorf("invalid card number length")
		}

		hash := sha256.Sum256([]byte(req.PAN))
		fingerprint := hex.EncodeToString(hash[:])
		encryptedPAN, err := crypto.Encrypt([]byte(req.PAN), s.vaultKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt card: %w", err)
		}
		lastFour := req.PAN[len(req.PAN)-4:]

		query := `
			INSERT INTO payment_methods 
			(id, merchant_id, type, card_last_four, card_exp_month, card_exp_year, card_fingerprint, encrypted_pan, encryption_key_id)
			VALUES ($1, $2, 'card', $3, $4, $5, $6, $7, 'v1')
		`
		_, err = s.db.Pool.Exec(ctx, query,
			tokenID, req.MerchantID, lastFour, req.ExpMonth, req.ExpYear, fingerprint, encryptedPAN)

		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		return &TokenizeResponse{TokenID: tokenID, LastFour: lastFour}, nil
	}

	if req.Type == "upi" {
		if req.UpiVpa == "" {
			return nil, fmt.Errorf("upi_vpa is required for UPI")
		}
		query := `INSERT INTO payment_methods (id, merchant_id, type, upi_vpa) VALUES ($1, $2, 'upi', $3)`
		_, err := s.db.Pool.Exec(ctx, query, tokenID, req.MerchantID, req.UpiVpa)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		return &TokenizeResponse{TokenID: tokenID}, nil
	}

	if req.Type == "netbanking" {
		if req.BankCode == "" {
			return nil, fmt.Errorf("bank_code is required for Netbanking")
		}
		query := `INSERT INTO payment_methods (id, merchant_id, type, bank_code) VALUES ($1, $2, 'netbanking', $3)`
		_, err := s.db.Pool.Exec(ctx, query, tokenID, req.MerchantID, req.BankCode)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		return &TokenizeResponse{TokenID: tokenID}, nil
	}

	return nil, fmt.Errorf("unsupported payment method type: %s", req.Type)
}
