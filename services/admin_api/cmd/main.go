package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"payment-processor/internal/config"
	"payment-processor/internal/database"
	"payment-processor/services/admin_api/handler"
	"payment-processor/services/admin_api/service"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect Database
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Services
	adminSvc := service.NewAdminService(db)
	adminHandler := handler.NewAdminHandler(adminSvc)

	// Setup Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/v1/admin/payments", adminHandler.GetPayments)
	r.Get("/v1/admin/ledger", adminHandler.GetLedger)
	r.Get("/v1/admin/subscriptions", adminHandler.GetSubscriptions)
	r.Get("/v1/admin/settlements", adminHandler.GetSettlements)
	r.Get("/v1/admin/merchant/settings", adminHandler.GetMerchantSettings)
	r.Put("/v1/admin/merchant/settings", adminHandler.UpdateMerchantSettings)
	r.Get("/v1/admin/reports/settlements", adminHandler.DownloadSettlementsReport)
	r.Post("/v1/admin/refunds", adminHandler.IssueRefund)
	r.Get("/v1/admin/refunds", adminHandler.GetRefunds)

	// Start Server
	server := &http.Server{
		Addr:    ":8084",
		Handler: r,
	}

	go func() {
		log.Println("Admin API service starting on port 8084...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Admin API service...")
	ctxShutDown, cancelShutDown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutDown()

	if err := server.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Admin API service exited cleanly")
}
