package stream

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerCreateStream(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	body := `{
		"name": "Camera 1",
		"url": "rtsp://localhost:8554/camera-1"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			response.Code,
		)
	}

	if !strings.Contains(response.Body.String(), `"name":"Camera 1"`) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}

	if !strings.Contains(
		response.Body.String(),
		`"state":"created"`,
	) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}
}

func TestHandlerCreateRejectsInvalidJSON(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(`{"name":`),
	)

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestHandlerCreateRejectsUnknownFields(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	body := `{
		"name": "Camera 1",
		"url": "rtsp://localhost:8554/camera-1",
		"unexpected": true
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(body),
	)

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestHandlerCreateRejectsInvalidStream(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	body := `{
		"name": "Camera 1",
		"url": "https://example.com/video"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(body),
	)

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestHandlerListStreams(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	first, err := service.Create(
		"Camera 1",
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("create first stream: %v", err)
	}

	second, err := service.Create(
		"Camera 2",
		"rtsp://localhost:8554/camera-2",
	)
	if err != nil {
		t.Fatalf("create second stream: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/streams",
		nil,
	)

	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	var got []Stream

	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(got))
	}

	if got[0].ID != first.ID {
		t.Fatalf(
			"expected first stream ID %q, got %q",
			first.ID,
			got[0].ID,
		)
	}

	if got[1].ID != second.ID {
		t.Fatalf(
			"expected second stream ID %q, got %q",
			second.ID,
			got[1].ID,
		)
	}
}

func TestHandlerGetStream(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	created, err := service.Create(
		"Camera 1",
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/streams/"+created.ID,
		nil,
	)
	request.SetPathValue("id", created.ID)

	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	var got Stream

	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf(
			"expected stream ID %q, got %q",
			created.ID,
			got.ID,
		)
	}
}

func TestHandlerGetStreamNotFound(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/streams/missing",
		nil,
	)
	request.SetPathValue("id", "missing")

	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.Code,
		)
	}
}

func TestHandlerDeleteStream(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	created, err := service.Create(
		"Camera 1",
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/streams/"+created.ID,
		nil,
	)
	request.SetPathValue("id", created.ID)

	response := httptest.NewRecorder()

	handler.Delete(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			response.Code,
		)
	}

	_, err = registry.Get(created.ID)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf(
			"expected stream to be deleted, got %v",
			err,
		)
	}
}

func TestHandlerDeleteStreamNotFound(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/streams/missing",
		nil,
	)
	request.SetPathValue("id", "missing")

	response := httptest.NewRecorder()

	handler.Delete(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.Code,
		)
	}
}

func TestHandlerCreateRejectsBlockedRTSPURL(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	policy := &rejectingRTSPURLPolicy{}

	service := NewServiceWithURLPolicy(
		registry,
		policy,
	)

	handler := NewHandler(service)

	body := `{
		"name": "Internal Camera",
		"url": "rtsp://10.0.0.10/live"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	response := httptest.NewRecorder()

	handler.Create(
		response,
		request,
	)

	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}

	if !strings.Contains(
		response.Body.String(),
		ErrRTSPURLNotAllowed.Error(),
	) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}

	if got := registry.List(); len(got) != 0 {
		t.Fatalf(
			"expected rejected stream not to be stored, got %d streams",
			len(got),
		)
	}
}

func TestHandlerCreateRejectsOversizedRequestBody(
	t *testing.T,
) {
	registry := NewMemoryRegistry()
	service := NewService(registry)
	handler := NewHandler(service)

	body := `{"name":"` +
		strings.Repeat("a", 5000) +
		`","url":"rtsp://camera.example.com/live"}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(body),
	)

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	response := httptest.NewRecorder()

	handler.Create(
		response,
		request,
	)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusRequestEntityTooLarge,
			response.Code,
		)
	}

	if !strings.Contains(
		response.Body.String(),
		"request body too large",
	) {
		t.Fatalf(
			"unexpected response body: %s",
			response.Body.String(),
		)
	}

	if got := registry.List(); len(got) != 0 {
		t.Fatalf(
			"expected oversized request not to be stored, got %d streams",
			len(got),
		)
	}
}
