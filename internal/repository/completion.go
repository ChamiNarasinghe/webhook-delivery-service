package repository

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CompleteAttempt records a delivery attempt and updates the event status.
func CompleteAttempt(
	ctx context.Context,
	db *pgxpool.Pool,
	eventID string,
	statusCode *int,
	responseBody string,
	deliveryError string,
	duration time.Duration,
	retryAfter string,
) error {

	id, err := uuid.Parse(eventID)
	if err != nil {
		return err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var deadline time.Time

	err = tx.QueryRow(
		ctx,
		`SELECT deadline_at FROM events WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&deadline)

	if err != nil {
		return err
	}

	var previousAttempts int

	err = tx.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM delivery_attempts WHERE event_id = $1`,
		id,
	).Scan(&previousAttempts)

	if err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO delivery_attempts (
			event_id, attempted_at, status_code,
			response_body, error, duration_ms
		)
		VALUES ($1, NOW(), $2, $3, $4, $5)
		`,
		id,
		statusCode,
		responseBody,
		deliveryError,
		duration.Milliseconds(),
	)

	if err != nil {
		return err
	}

	success := statusCode != nil &&
		*statusCode >= 200 &&
		*statusCode < 300 &&
		deliveryError == ""

	if success {
		_, err = tx.Exec(
			ctx,
			`
			UPDATE events
SET status = 'delivered',
    delivered_at = NOW(),
    processing_started_at = NULL
WHERE id = $1
			`,
			id,
		)
	} else {
		now := time.Now().UTC()
		attemptNumber := previousAttempts + 1

		delaySeconds := 1 << min(attemptNumber-1, 12)

		if delaySeconds > 3600 {
			delaySeconds = 3600
		}

		retryDelay := time.Duration(delaySeconds) * time.Second

		// Use Retry-After when the receiver provides a valid value.
		if retryAfter != "" {
			if seconds, parseErr := strconv.Atoi(strings.TrimSpace(retryAfter)); parseErr == nil && seconds >= 0 {
				retryDelay = time.Duration(seconds) * time.Second
			} else if retryTime, parseErr := http.ParseTime(retryAfter); parseErr == nil {
				retryDelay = retryTime.Sub(now)
				if retryDelay < 0 {
					retryDelay = 0
				}
			}
		}

		nextAttempt := now.Add(retryDelay)

		if !nextAttempt.Before(deadline) {
			_, err = tx.Exec(
				ctx,
				`UPDATE events
SET status = 'dead',
    processing_started_at = NULL
WHERE id = $1`,
				id,
			)
		} else {
			_, err = tx.Exec(
				ctx,
				`
				UPDATE events
SET status = 'pending',
    next_attempt_at = $2,
    processing_started_at = NULL
WHERE id = $1
				`,
				id,
				nextAttempt,
			)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	return tx.Commit(ctx)
}
