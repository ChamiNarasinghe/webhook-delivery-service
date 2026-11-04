package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"webhook-delivery-service/internal/models"
	"webhook-delivery-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"webhook-delivery-service/internal/config"
)

type EventHandler struct {
	DB      *pgxpool.Pool
	Tenants *config.TenantConfig
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {

	// Only allow POST requests.
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the incoming JSON request.
	var request models.CreateEventRequest

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Reject additional JSON values after the first object.
	var extra interface{}

	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			http.Error(w, "Request must contain only one JSON object", http.StatusBadRequest)
		} else {
			http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		}
		return
	}

	// Validate required fields.
	request.TenantID = strings.TrimSpace(request.TenantID)
	request.Type = strings.TrimSpace(request.Type)

	if request.TenantID == "" || request.Type == "" || len(request.Payload) == 0 {
		http.Error(w, "tenant_id, type and payload are required", http.StatusBadRequest)
		return
	}

	// Check whether the tenant exists in our configuration.
	if _, exists := h.Tenants.Tenants[request.TenantID]; !exists {
		http.Error(w, "Unknown tenant ID", http.StatusBadRequest)
		return
	}

	if !json.Valid(request.Payload) {
		http.Error(w, "Payload must be valid JSON", http.StatusBadRequest)
		return
	}

	// Save the event in PostgreSQL.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	eventID, err := repository.Create(
		ctx,
		h.DB,
		request.TenantID,
		request.Type,
		request.Payload,
	)

	if err != nil {
		http.Error(w, "Failed to save event", http.StatusInternalServerError)
		return
	}

	// Return the created event ID.
	response := map[string]interface{}{
		"id":      eventID,
		"status":  "pending",
		"message": "Event accepted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// GetEvent retrieves an event and its delivery attempt history.
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventID := r.PathValue("id")

	// Validate the event ID.
	if _, err := uuid.Parse(eventID); err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	event, attempts, err := repository.GetByID(
		ctx,
		h.DB,
		eventID,
	)

	if err != nil {
		http.Error(w, "Failed to retrieve event", http.StatusInternalServerError)
		return
	}

	if event == nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"event":    event,
		"attempts": attempts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDeadLetters returns all events that have permanently failed.
func (h *EventHandler) GetDeadLetters(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	events, err := repository.GetDeadLetters(ctx, h.DB)
	if err != nil {
		http.Error(w, "Failed to retrieve dead-letter events", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"count":  len(events),
		"events": events,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
