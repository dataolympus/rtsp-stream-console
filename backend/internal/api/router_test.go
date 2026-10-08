package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/stream"
)

type fakeRuntimeController struct {
	startID string
	stopID  string
}

func (f *fakeRuntimeController) Start(id string) error {
	f.startID = id
	return nil
}

func (f *fakeRuntimeController) Stop(id string) error {
	f.stopID = id
	return nil
}

func newTestRouterWithHub() (
	http.Handler,
	*stream.Hub,
) {
	healthHandler := health.New(
		func() bool { return true },
	)

	registry := stream.NewMemoryRegistry()
	service := stream.NewService(registry)
	streamHandler := stream.NewHandler(service)

	hub := stream.NewHub()
	websocketHandler :=
		stream.NewWebSocketHandler(hub)

	runtimeController :=
		&fakeRuntimeController{}

	runtimeHandler :=
		stream.NewRuntimeHandler(
			runtimeController,
		)

	return NewRouter(
		healthHandler,
		streamHandler,
		websocketHandler,
		runtimeHandler,
	), hub
}

func newTestRouter() http.Handler {
	router, _ := newTestRouterWithHub()

	return router
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

func TestRouterRoutesStreamWebSocket(t *testing.T) {
	router, hub := newTestRouterWithHub()

	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	wsURL := "ws" +
		strings.TrimPrefix(
			server.URL,
			"http",
		) +
		"/api/v1/streams/stream-1/ws"

	conn, _, err := websocket.Dial(
		ctx,
		wsURL,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"dial websocket: %v",
			err,
		)
	}
	defer conn.CloseNow()

	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		if hub.SubscriberCount("stream-1") == 1 {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if got := hub.SubscriberCount("stream-1"); got != 1 {
		t.Fatalf(
			"expected 1 subscriber, got %d",
			got,
		)
	}

	hub.Publish(
		"stream-1",
		[]byte("router-media"),
	)

	messageType, payload, err :=
		conn.Read(ctx)
	if err != nil {
		t.Fatalf(
			"read websocket message: %v",
			err,
		)
	}

	if messageType != websocket.MessageBinary {
		t.Fatalf(
			"expected binary message, got %v",
			messageType,
		)
	}

	if string(payload) != "router-media" {
		t.Fatalf(
			"expected payload %q, got %q",
			"router-media",
			payload,
		)
	}
}

func TestRouterRoutesStreamStart(t *testing.T) {
	controller := &fakeRuntimeController{}

	runtimeHandler :=
		stream.NewRuntimeHandler(controller)

	healthHandler := health.New(
		func() bool { return true },
	)

	registry := stream.NewMemoryRegistry()
	service := stream.NewService(registry)
	streamHandler := stream.NewHandler(service)

	hub := stream.NewHub()
	websocketHandler :=
		stream.NewWebSocketHandler(hub)

	router := NewRouter(
		healthHandler,
		streamHandler,
		websocketHandler,
		runtimeHandler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/stream-1/start",
		nil,
	)

	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			response.Code,
		)
	}

	if controller.startID != "stream-1" {
		t.Fatalf(
			"expected start ID %q, got %q",
			"stream-1",
			controller.startID,
		)
	}
}

func TestRouterRoutesStreamStop(t *testing.T) {
	controller := &fakeRuntimeController{}

	runtimeHandler :=
		stream.NewRuntimeHandler(controller)

	healthHandler := health.New(
		func() bool { return true },
	)

	registry := stream.NewMemoryRegistry()
	service := stream.NewService(registry)
	streamHandler := stream.NewHandler(service)

	hub := stream.NewHub()
	websocketHandler :=
		stream.NewWebSocketHandler(hub)

	router := NewRouter(
		healthHandler,
		streamHandler,
		websocketHandler,
		runtimeHandler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/stream-1/stop",
		nil,
	)

	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			response.Code,
		)
	}

	if controller.stopID != "stream-1" {
		t.Fatalf(
			"expected stop ID %q, got %q",
			"stream-1",
			controller.stopID,
		)
	}
}
