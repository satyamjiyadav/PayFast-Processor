package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"payment-processor/services/gateway/service"
)

// ResponseRecorder captures the response status and body
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
	Body       *bytes.Buffer
}

func (rw *ResponseRecorder) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseRecorder) Write(b []byte) (int, error) {
	rw.Body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Idempotency middleware ensures safe retries using Redis & DB
func Idempotency(svc *service.IdempotencyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			idempotencyKey := r.Header.Get("Idempotency-Key")
			if idempotencyKey == "" {
				http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
				return
			}

			merchantID, ok := r.Context().Value(MerchantIDKey).(string)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// 1. Check if we already processed this request (DB Check)
			record, err := svc.GetRecord(r.Context(), merchantID, idempotencyKey)
			if err != nil {
				http.Error(w, "internal server error during idempotency check", http.StatusInternalServerError)
				return
			}

			if record != nil {
				if record.Status == "completed" {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Idempotent-Replayed", "true")
					w.WriteHeader(record.ResponseCode)
					w.Write(record.ResponseBody)
					return
				} else {
					// In progress
					http.Error(w, "request already in progress", http.StatusConflict)
					return
				}
			}

			// 2. Lock the key (Redis SetNX) to prevent thundering herd
			acquired, err := svc.AcquireLock(r.Context(), merchantID, idempotencyKey)
			if err != nil || !acquired {
				http.Error(w, "request already in progress", http.StatusConflict)
				return
			}
			defer svc.ReleaseLock(context.Background(), merchantID, idempotencyKey)

			// 3. Create Pending Record in DB
			err = svc.CreatePendingRecord(r.Context(), merchantID, idempotencyKey, r.URL.Path, r.Method)
			if err != nil {
				http.Error(w, "internal server error saving pending state", http.StatusInternalServerError)
				return
			}

			// 4. Capture the response
			recorder := &ResponseRecorder{
				ResponseWriter: w,
				StatusCode:     http.StatusOK,
				Body:           &bytes.Buffer{},
			}

			next.ServeHTTP(recorder, r)

			// 5. Complete Record in DB
			var responseBody interface{}
			// Try parsing as json to store properly
			json.Unmarshal(recorder.Body.Bytes(), &responseBody)
			if responseBody == nil {
				responseBody = map[string]string{"raw": recorder.Body.String()}
			}

			svc.CompleteRecord(context.Background(), merchantID, idempotencyKey, recorder.StatusCode, responseBody)
		})
	}
}
