package handler

import (
	"encoding/json"
	"net/http"

	"payment-processor/services/admin_api/service"
)

type AdminHandler struct {
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) GetPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := h.svc.GetPayments(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, payments)
}

func (h *AdminHandler) GetSettlements(w http.ResponseWriter, r *http.Request) {
	settlements, err := h.svc.GetSettlements(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, settlements)
}

func (h *AdminHandler) GetLedger(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.GetRecentLedgerEntries(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries)
}

func (h *AdminHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, err := h.svc.GetSubscriptions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, subs)
}

func (h *AdminHandler) GetMerchantSettings(w http.ResponseWriter, r *http.Request) {
	// Use actual local merchant ID
	settings, err := h.svc.GetMerchantSettings(r.Context(), "merch_01test")
	if err != nil {
		// fallback if merchant not found
		writeJSON(w, map[string]string{"payout_schedule": "instant"})
		return
	}
	writeJSON(w, settings)
}

func (h *AdminHandler) UpdateMerchantSettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PayoutSchedule string `json:"payout_schedule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Use actual local merchant ID
	err := h.svc.UpdateMerchantSettings(r.Context(), "merch_01test", payload.PayoutSchedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if data == nil {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(data)
}

func (h *AdminHandler) DownloadSettlementsReport(w http.ResponseWriter, r *http.Request) {
	merchantID := "merch_01test" // Hardcoded for demo/simplicity based on current architecture
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "24h"
	}

	csvData, err := h.svc.GenerateSettlementsReport(r.Context(), merchantID, timeRange)
	if err != nil {
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"settlements_report.csv\"")
	w.Write([]byte(csvData))
}

func (h *AdminHandler) IssueRefund(w http.ResponseWriter, r *http.Request) {
	merchantID := "merch_01test" // hardcoded for demo

	var req struct {
		PaymentID string `json:"payment_id"`
		Amount    int64  `json:"amount"` // using full amount if we do full refunds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	err := h.svc.IssueRefund(r.Context(), merchantID, req.PaymentID, req.Amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func (h *AdminHandler) GetRefunds(w http.ResponseWriter, r *http.Request) {
	merchantID := "merch_01test" // hardcoded for demo
	refunds, err := h.svc.GetRefunds(r.Context(), merchantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, refunds)
}
