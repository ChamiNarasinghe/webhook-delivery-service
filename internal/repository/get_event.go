package repository

import (
	"context"
	"errors"

	"webhook-delivery-service/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetByID retrieves an event and all its delivery attempts.
func GetByID(
	ctx context.Context,
	db *pgxpool.Pool,
	eventID string,
) (*models.Event, []models.DeliveryAttempt, error) {

	id, err := uuid.Parse(eventID)
	if err != nil {
		return nil, nil, err
	}

	var event models.Event
	var payload []byte

	err = db.QueryRow(
		ctx,
		`
		SELECT id, tenant_id, event_type, payload,
		       status, created_at, next_attempt_at,
		       deadline_at, delivered_at
		FROM events
		WHERE id = $1
		`,
		id,
	).Scan(
		&event.ID,
		&event.TenantID,
		&event.Type,
		&payload,
		&event.Status,
		&event.CreatedAt,
		&event.NextAttemptAt,
		&event.DeadlineAt,
		&event.DeliveredAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	event.Payload = payload

	rows, err := db.Query(
		ctx,
		`
		SELECT id, event_id, attempted_at,
		       COALESCE(status_code, 0),
		       COALESCE(response_body, ''),
		       COALESCE(error, ''),
		       duration_ms
		FROM delivery_attempts
		WHERE event_id = $1
		ORDER BY attempted_at ASC, id ASC
		`,
		id,
	)

	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	attempts := make([]models.DeliveryAttempt, 0)

	for rows.Next() {
		var attempt models.DeliveryAttempt

		err := rows.Scan(
			&attempt.ID,
			&attempt.EventID,
			&attempt.AttemptedAt,
			&attempt.StatusCode,
			&attempt.ResponseBody,
			&attempt.Error,
			&attempt.DurationMS,
		)

		if err != nil {
			return nil, nil, err
		}

		attempts = append(attempts, attempt)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return &event, attempts, nil
}
