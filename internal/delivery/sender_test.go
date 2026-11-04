
package delivery

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendWebhook(t *testing.T) {
	payload := []byte(`{"payment_id":12345,"amount":5000}`)
	secret := "acme-secret"
	eventID := "test-event-123"

	// Create a temporary receiver.
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			// Check HTTP method.
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			// Check event ID header.
			if r.Header.Get("X-Event-ID") != eventID {
				t.Errorf("incorrect event ID")
			}

			// Check signature.
			expectedSignature := GenerateSignature(payload, secret)

			if r.Header.Get("X-Webhook-Signature") != expectedSignature {
				t.Errorf("incorrect webhook signature")
			}

			// Check payload.
			body, _ := io.ReadAll(r.Body)

			if string(body) != string(payload) {
				t.Errorf("incorrect payload")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Event received"))
		},
	))

	defer server.Close()

	// Send the webhook.
	result, err := Send(
		context.Background(),
		server.URL,
		secret,
		eventID,
		payload,
	)

	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}

	if !strings.Contains(result.ResponseBody, "Event received") {
		t.Fatalf("unexpected response body: %s", result.ResponseBody)
	}

	if result.Duration <= 0 {
		t.Fatal("expected positive request duration")
	}
}
