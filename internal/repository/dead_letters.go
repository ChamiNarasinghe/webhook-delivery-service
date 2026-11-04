package repository

import (
	"context"

	"webhook-delivery-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetDeadLetters retrieves all events that have permanently failed.
func GetDeadLetters(
	ctx context.Context,
	db *pgxpool.Pool,
) ([]models.Event, error) {

	query := `
		SELECT id, tenant_id, event_type, payload,
		       status, created_at, next_attempt_at,
		       deadline_at, delivered_at
		FROM events
		WHERE status = 'dead'
		ORDER BY created_at DESC
	`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.Event, 0)

	for rows.Next() {
		var event models.Event
		var payload []byte

		err := rows.Scan(
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

		if err != nil {
			return nil, err
		}

		event.Payload = payload
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
