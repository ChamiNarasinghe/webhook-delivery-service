package delivery

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type DeliveryResult struct {
	StatusCode   int
	ResponseBody string
	Duration     time.Duration
	RetryAfter   string
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// Send sends an event to a tenant's webhook endpoint.
func Send(
	ctx context.Context,
	webhookURL string,
	secret string,
	eventID string,
	payload []byte,
) (*DeliveryResult, error) {

	startTime := time.Now()

	// Generate the HMAC signature.
	signature := GenerateSignature(payload, secret)

	// Create an HTTP POST request.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		webhookURL,
		strings.NewReader(string(payload)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add required headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-ID", eventID)
	req.Header.Set("X-Webhook-Signature", signature)

	// Send the request.
	resp, err := httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read a limited amount of the response body.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &DeliveryResult{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
		Duration:     time.Since(startTime),
		RetryAfter:   resp.Header.Get("Retry-After"),
	}
	return result, nil
}
