
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Create saves a new webhook event in PostgreSQL.
func Create(
	ctx context.Context,
	db *pgxpool.Pool,
	tenantID string,
	eventType string,
	payload json.RawMessage,
) (uuid.UUID, error) {

	eventID := uuid.New()

	now := time.Now().UTC()
	deadline := now.Add(24 * time.Hour)

	query := `
		INSERT INTO events (
			id,
			tenant_id,
			event_type,
			payload,
			status,
			created_at,
			next_attempt_at,
			deadline_at
		)
		VALUES ($1, $2, $3, $4, 'pending', $5, $5, $6)
	`

	_, err := db.Exec(
		ctx,
		query,
		eventID,
		tenantID,
		eventType,
		[]byte(payload),
		now,
		deadline,
	)

	if err != nil {
		return uuid.Nil, err
	}

	return eventID, nil
}
