
package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"webhook-delivery-service/internal/config"
)

func newTestHandler() *EventHandler {
	return &EventHandler{
		Tenants: &config.TenantConfig{
			Tenants: map[string]config.Tenant{
				"acme": {},
			},
		},
	}
}

func TestCreateEventValidation(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "reject GET request",
			method:         http.MethodGet,
			body:           `{}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "reject invalid JSON",
			method:         http.MethodPost,
			body:           `invalid json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "reject unknown JSON field",
			method:         http.MethodPost,
			body:           `{"tenant_id":"acme","type":"test","payload":{},"extra":"value"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "reject multiple JSON objects",
			method:         http.MethodPost,
			body:           `{"tenant_id":"acme","type":"test","payload":{}} {}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "reject missing required fields",
			method:         http.MethodPost,
			body:           `{"tenant_id":"acme","type":"","payload":{}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "reject unknown tenant",
			method:         http.MethodPost,
			body:           `{"tenant_id":"unknown","type":"test","payload":{}}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := newTestHandler()

			request := httptest.NewRequest(
				test.method,
				"/events",
				strings.NewReader(test.body),
			)

			recorder := httptest.NewRecorder()

			handler.CreateEvent(recorder, request)

			if recorder.Code != test.expectedStatus {
				t.Errorf(
					"expected status %d, got %d",
					test.expectedStatus,
					recorder.Code,
				)
			}
		})
	}
}

func TestGetEventValidation(t *testing.T) {
	t.Run("reject POST request", func(t *testing.T) {
		handler := newTestHandler()

		request := httptest.NewRequest(http.MethodPost, "/events/test", nil)
		recorder := httptest.NewRecorder()

		handler.GetEvent(recorder, request)

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf(
				"expected status %d, got %d",
				http.StatusMethodNotAllowed,
				recorder.Code,
			)
		}
	})

	t.Run("reject invalid event ID", func(t *testing.T) {
		handler := newTestHandler()

		request := httptest.NewRequest(http.MethodGet, "/events/invalid", nil)
		request.SetPathValue("id", "invalid")

		recorder := httptest.NewRecorder()

		handler.GetEvent(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf(
				"expected status %d, got %d",
				http.StatusBadRequest,
				recorder.Code,
			)
		}
	})
}

func TestGetDeadLettersMethod(t *testing.T) {
	handler := newTestHandler()

	request := httptest.NewRequest(http.MethodPost, "/dead-letters", nil)
	recorder := httptest.NewRecorder()

	handler.GetDeadLetters(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}
