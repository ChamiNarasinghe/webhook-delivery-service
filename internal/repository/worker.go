package repository

import (
	"context"
	"errors"

	"webhook-delivery-service/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClaimNext claims the next available event for a tenant.
func ClaimNext(
	ctx context.Context,
	db *pgxpool.Pool,
	tenantID string,
) (*models.Event, error) {

	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Prevent two workers from processing the same tenant simultaneously.
	_, err = tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		tenantID,
	)

	if err != nil {
		return nil, err
	}

	// Mark expired events as dead letters.
	// Events must not be retried after their 24-hour deadline.
	_, err = tx.Exec(
		ctx,
		`
    UPDATE events
    SET status = 'dead',
        processing_started_at = NULL
    WHERE tenant_id = $1
      AND status IN ('pending', 'processing')
      AND deadline_at <= NOW()
    `,
		tenantID,
	)

	if err != nil {
		return nil, err
	}

	// Recover events abandoned by workers that crashed.

	_, err = tx.Exec(
		ctx,
		`
    UPDATE events
    SET status = 'pending',
        processing_started_at = NULL,
        next_attempt_at = NOW()
    WHERE tenant_id = $1
      AND status = 'processing'
      AND (
          processing_started_at IS NULL
          OR processing_started_at < NOW() - INTERVAL '30 seconds'
      )
    `,
		tenantID,
	)

	if err != nil {
		return nil, err
	}

	// Find the earliest event that is ready for delivery.
	// Earlier unfinished events block later events.
	query := `
		SELECT id, tenant_id, event_type, payload,
		       status, created_at, next_attempt_at,
		       deadline_at, delivered_at
		FROM events e
		WHERE e.tenant_id = $1
		  AND e.status = 'pending'
		  AND e.next_attempt_at <= NOW()
		  AND NOT EXISTS (
		      SELECT 1
		      FROM events earlier
		      WHERE earlier.tenant_id = e.tenant_id
		        AND (
    earlier.created_at < e.created_at
    OR (
        earlier.created_at = e.created_at
        AND earlier.id < e.id
    )
)
		        AND earlier.status NOT IN ('delivered', 'dead')
		  )
		ORDER BY e.created_at, e.id
		LIMIT 1
		FOR UPDATE
	`

	var event models.Event
	var eventID uuid.UUID
	var payload []byte

	err = tx.QueryRow(ctx, query, tenantID).Scan(
		&eventID,
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
		return nil, tx.Commit(ctx)
	}

	if err != nil {
		return nil, err
	}

	event.ID = eventID.String()
	event.Payload = payload

	// Mark the event as being processed.
	_, err = tx.Exec(
		ctx,
		`UPDATE events
     SET status = 'processing',
     processing_started_at = NOW()
     WHERE id = $1`,
		eventID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	event.Status = "processing"

	return &event, nil
}
