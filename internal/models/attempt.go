package models

import "time"

// DeliveryAttempt represents one attempt to deliver a webhook event.
type DeliveryAttempt struct {
	ID           int64     `json:"id"`
	EventID      string    `json:"event_id"`
	AttemptedAt  time.Time `json:"attempted_at"`
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `json:"response_body"`
	Error        string    `json:"error"`
	DurationMS   int64     `json:"duration_ms"`
}
