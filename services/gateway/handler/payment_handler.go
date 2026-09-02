package handler

import (
	"encoding/json"
	"net/http"

	"payment-processor/services/gateway/middleware"
	"payment-processor/services/gateway/service"
)

type PaymentHandler struct {
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := r.Context().Value(middleware.MerchantIDKey).(string)
	if !ok || merchantID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req service.CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	resp, err := h.svc.ProcessPayment(r.Context(), merchantID, idempotencyKey, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
