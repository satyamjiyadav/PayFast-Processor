package handler

import (
	"encoding/json"
	"net/http"

	"payment-processor/services/subscriptions/service"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req service.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	// Assuming a dummy auth middleware injected merchant_id
	// req.MerchantID = r.Context().Value(middleware.MerchantIDKey).(string)
	if req.MerchantID == "" {
		req.MerchantID = "merch_01test" // fallback for demo
	}

	sub, err := h.svc.CreateSubscription(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}
