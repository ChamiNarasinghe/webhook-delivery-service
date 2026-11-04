
package models

import (
	"encoding/json"
	"time"
)

// Event represents a webhook event stored in our database.
type Event struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	NextAttemptAt time.Time       `json:"next_attempt_at"`
	DeadlineAt    time.Time       `json:"deadline_at"`
	DeliveredAt   *time.Time      `json:"delivered_at,omitempty"`
}

// CreateEventRequest represents the data received from the client.
type CreateEventRequest struct {
	TenantID string          `json:"tenant_id"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}
