package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/stream"
)

func newTestRouter() http.Handler {
	healthHandler := health.New(func() bool {
		return true
	})

	registry := stream.NewMemoryRegistry()
	service := stream.NewService(registry)
	streamHandler := stream.NewHandler(service)

	return NewRouter(
		healthHandler,
		streamHandler,
	)
}

func TestRouterCreateAndListStreams(t *testing.T) {
	router := newTestRouter()

	createBody := `{
		"name": "Camera 1",
		"url": "rtsp://localhost:8554/camera-1"
	}`

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(createBody),
	)
	createRequest.Header.Set("Content-Type", "application/json")

	createResponse := httptest.NewRecorder()

	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf(
			"expected create status %d, got %d",
			http.StatusCreated,
			createResponse.Code,
		)
	}

	listRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/streams",
		nil,
	)

	listResponse := httptest.NewRecorder()

	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf(
			"expected list status %d, got %d",
			http.StatusOK,
			listResponse.Code,
		)
	}

	var got []stream.Stream

	if err := json.NewDecoder(listResponse.Body).Decode(&got); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 stream, got %d", len(got))
	}

	if got[0].Name != "Camera 1" {
		t.Fatalf(
			"expected stream name %q, got %q",
			"Camera 1",
			got[0].Name,
		)
	}
}

func TestRouterRoutesStreamID(t *testing.T) {
	router := newTestRouter()

	createBody := `{
		"name": "Camera 1",
		"url": "rtsp://localhost:8554/camera-1"
	}`

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(createBody),
	)

	createResponse := httptest.NewRecorder()

	router.ServeHTTP(createResponse, createRequest)

	var created stream.Stream

	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/streams/"+created.ID,
		nil,
	)

	getResponse := httptest.NewRecorder()

	router.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf(
			"expected get status %d, got %d",
			http.StatusOK,
			getResponse.Code,
		)
	}

	var got stream.Stream

	if err := json.NewDecoder(getResponse.Body).Decode(&got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf(
			"expected stream ID %q, got %q",
			created.ID,
			got.ID,
		)
	}
}

func TestRouterDeleteStream(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		strings.NewReader(`{
			"name": "Camera 1",
			"url": "rtsp://localhost:8554/camera-1"
		}`),
	)

	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, createRequest)

	var created stream.Stream

	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	deleteRequest := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/streams/"+created.ID,
		nil,
	)

	deleteResponse := httptest.NewRecorder()

	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf(
			"expected delete status %d, got %d",
			http.StatusNoContent,
			deleteResponse.Code,
		)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/streams/"+created.ID,
		nil,
	)

	getResponse := httptest.NewRecorder()

	router.ServeHTTP(getResponse, getRequest)

	if getResponse.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d after deletion, got %d",
			http.StatusNotFound,
			getResponse.Code,
		)
	}
}
