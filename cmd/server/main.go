package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webhook-delivery-service/internal/config"
	"webhook-delivery-service/internal/database"
	"webhook-delivery-service/internal/handlers"
	"webhook-delivery-service/internal/worker"
)

func main() {
	log.Println("Starting Webhook Delivery Service...")

	// Connect to PostgreSQL.
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	log.Println("Connected to PostgreSQL")

	// Create database tables if they do not exist.
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// Load tenant webhook URLs and secrets.
	tenantConfig, err := config.LoadTenants(
		"internal/config/tenants.json",
	)
	if err != nil {
		log.Fatalf("Tenant configuration failed: %v", err)
	}

	log.Printf("Loaded %d tenants", len(tenantConfig.Tenants))

	// Create a context that stops workers during shutdown.
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	// Start independent delivery workers for each tenant.
	worker.Start(ctx, db, tenantConfig)

	// Set up HTTP API.
	eventHandler := &handlers.EventHandler{
		DB:      db,
		Tenants: tenantConfig,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/events", eventHandler.CreateEvent)
	mux.HandleFunc("GET /events/{id}", eventHandler.GetEvent)
	mux.HandleFunc("GET /dead-letters", eventHandler.GetDeadLetters)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the HTTP server.
	serverErrors := make(chan error, 1)

	go func() {
		log.Println("Webhook Delivery Service running on :8080")
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for either a shutdown signal or a server error.
	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received")

	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}

	// Stop workers and shut down the HTTP server gracefully.
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Webhook Delivery Service stopped")
}
