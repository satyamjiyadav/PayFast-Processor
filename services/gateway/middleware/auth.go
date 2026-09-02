package middleware

import (
	"context"
	"net/http"
	"strings"

	"payment-processor/internal/database"
)

type contextKey string

const MerchantIDKey contextKey = "merchant_id"

// APIAuth middleware validates the Authorization header (Bearer sk_test_...)
func APIAuth(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}
			apiKey := parts[1]

			// In a real production system, you would hash the API key and look it up.
			// For this local demo, we'll do a simple mock check against our test merchant.
			// The dummy merchant was inserted with id 'merch_01test' and api_key_prefix 'sk_test_'
			if !strings.HasPrefix(apiKey, "sk_test_") {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}

			merchantID := "merch_01test"

			// Pass merchant ID into context
			ctx := context.WithValue(r.Context(), MerchantIDKey, merchantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
