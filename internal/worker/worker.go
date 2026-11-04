package worker

import (
	"context"
	"log"
	"time"

	"webhook-delivery-service/internal/config"
	"webhook-delivery-service/internal/delivery"
	"webhook-delivery-service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Start starts one independent worker for each configured tenant.
func Start(
	ctx context.Context,
	db *pgxpool.Pool,
	tenantConfig *config.TenantConfig,
) {
	for tenantID, tenant := range tenantConfig.Tenants {
		go runTenantWorker(ctx, db, tenantID, tenant)
		log.Printf("Started delivery worker for tenant: %s", tenantID)
	}
}

// runTenantWorker processes events sequentially for one tenant.
func runTenantWorker(
	ctx context.Context,
	db *pgxpool.Pool,
	tenantID string,
	tenant config.Tenant,
) {
	for {
		if ctx.Err() != nil {
			return
		}

		event, err := repository.ClaimNext(ctx, db, tenantID)

		if err != nil {
			log.Printf("Worker error for tenant %s: %v", tenantID, err)
			if !wait(ctx, 2*time.Second) {
				return
			}
			continue
		}

		// No event is ready. Check again after one second.
		if event == nil {
			if !wait(ctx, time.Second) {
				return
			}
			continue
		}

		// Deliver the event to the tenant's webhook.
		start := time.Now()

		result, sendErr := delivery.Send(
			ctx,
			tenant.WebhookURL,
			tenant.Secret,
			event.ID,
			event.Payload,
		)

		duration := time.Since(start)

		var statusCode *int
		var responseBody string
		var deliveryError string
		var retryAfter string

		if sendErr != nil {
			deliveryError = sendErr.Error()
		} else {
			statusCode = &result.StatusCode
			responseBody = result.ResponseBody
			retryAfter = result.RetryAfter
			duration = result.Duration
		}

		// Save the attempt and update the event status.
		err = repository.CompleteAttempt(
			ctx,
			db,
			event.ID,
			statusCode,
			responseBody,
			deliveryError,
			duration,
			retryAfter,
		)

		if err != nil {
			log.Printf(
				"Failed to complete event %s for tenant %s: %v",
				event.ID,
				tenantID,
				err,
			)
		}
	}
}

// wait pauses the worker but allows it to stop when its context is cancelled.
func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
