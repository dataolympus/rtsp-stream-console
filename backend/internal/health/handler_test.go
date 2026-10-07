package health

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	handler := New(nil)

	request := httptest.NewRequest(
		http.MethodGet,
		"/healthz",
		nil,
	)

	response := httptest.NewRecorder()

	handler.Healthz(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	if !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}
}

func TestReadyzWhenReady(t *testing.T) {
	handler := New(func() bool {
		return true
	})

	request := httptest.NewRequest(
		http.MethodGet,
		"/readyz",
		nil,
	)

	response := httptest.NewRecorder()

	handler.Readyz(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}
}

func TestReadyzWhenNotReady(t *testing.T) {
	handler := New(func() bool {
		return false
	})

	request := httptest.NewRequest(
		http.MethodGet,
		"/readyz",
		nil,
	)

	response := httptest.NewRecorder()

	handler.Readyz(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			response.Code,
		)
	}
}
