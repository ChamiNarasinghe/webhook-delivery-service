
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecordAttempt saves one webhook delivery attempt.
func RecordAttempt(
	ctx context.Context,
	db *pgxpool.Pool,
	eventID string,
	statusCode *int,
	responseBody string,
	deliveryError string,
	duration time.Duration,
) error {

	id, err := uuid.Parse(eventID)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO delivery_attempts (
			event_id,
			attempted_at,
			status_code,
			response_body,
			error,
			duration_ms
		)
		VALUES ($1, NOW(), $2, $3, $4, $5)
	`

	_, err = db.Exec(
		ctx,
		query,
		id,
		statusCode,
		responseBody,
		deliveryError,
		duration.Milliseconds(),
	)

	return err
}
