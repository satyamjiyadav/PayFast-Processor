package handler

import (
	"encoding/json"
	"net/http"

	"payment-processor/services/vault/service"
)

type TokenHandler struct {
	svc *service.VaultService
}

func NewTokenHandler(svc *service.VaultService) *TokenHandler {
	return &TokenHandler{svc: svc}
}

func (h *TokenHandler) Tokenize(w http.ResponseWriter, r *http.Request) {
	var req service.TokenizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.TokenizeCard(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
