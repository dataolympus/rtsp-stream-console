package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/stream"
)

func TestStreamingEndToEnd(t *testing.T) {
	sourceURL := os.Getenv("RTSP_INTEGRATION_URL")
	if sourceURL == "" {
		t.Skip("RTSP_INTEGRATION_URL is not set")
	}

	appCtx, cancelApp := context.WithCancel(
		context.Background(),
	)
	defer cancelApp()

	registry := stream.NewMemoryRegistry()

	service := stream.NewService(registry)
	streamHandler := stream.NewHandler(service)

	hub := stream.NewHub()
	pump := stream.NewMediaPump(hub)

	runner := stream.NewFFmpegRunner("ffmpeg")

	manager := stream.NewManager(
		appCtx,
		registry,
		runner,
		pump,
	)

	runtimeHandler :=
		stream.NewRuntimeHandler(manager)

	websocketHandler :=
		stream.NewWebSocketHandler(hub)

	healthHandler := health.New(
		func() bool { return true },
	)

	router := NewRouter(
		healthHandler,
		streamHandler,
		websocketHandler,
		runtimeHandler,
	)

	server := httptest.NewServer(router)
	defer server.Close()

	created := createIntegrationStream(
		t,
		server.URL,
		sourceURL,
	)

	// Subscribe before starting the runtime so the test does not
	// miss the beginning of the MPEG-TS byte stream.
	wsURL := "ws" +
		strings.TrimPrefix(
			server.URL,
			"http",
		) +
		"/api/v1/streams/" +
		created.ID +
		"/ws"

	wsCtx, cancelWS := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelWS()

	conn, _, err := websocket.Dial(
		wsCtx,
		wsURL,
		nil,
	)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.CloseNow()

	waitForSubscribers(
		t,
		hub,
		created.ID,
		1,
	)

	postIntegrationCommand(
		t,
		server.URL+
			"/api/v1/streams/"+
			created.ID+
			"/start",
	)

	waitForIntegrationState(
		t,
		server.URL,
		created.ID,
		stream.StateLive,
	)

	media := readMPEGTSPayload(
		t,
		wsCtx,
		conn,
	)

	if !containsMPEGTSSync(media) {
		t.Fatalf(
			"received %d bytes but could not find MPEG-TS sync pattern",
			len(media),
		)
	}

	postIntegrationCommand(
		t,
		server.URL+
			"/api/v1/streams/"+
			created.ID+
			"/stop",
	)

	waitForIntegrationState(
		t,
		server.URL,
		created.ID,
		stream.StateStopped,
	)
}

func createIntegrationStream(
	t *testing.T,
	baseURL string,
	sourceURL string,
) stream.Stream {
	t.Helper()

	body, err := json.Marshal(
		map[string]string{
			"name": "Integration Camera",
			"url":  sourceURL,
		},
	)
	if err != nil {
		t.Fatalf("marshal create request: %v", err)
	}

	response, err := http.Post(
		baseURL+"/api/v1/streams",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected create status %d, got %d: %s",
			http.StatusCreated,
			response.StatusCode,
			payload,
		)
	}

	var created stream.Stream

	if err := json.NewDecoder(
		response.Body,
	).Decode(&created); err != nil {
		t.Fatalf("decode created stream: %v", err)
	}

	return created
}

func postIntegrationCommand(
	t *testing.T,
	url string,
) {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodPost,
		url,
		nil,
	)
	if err != nil {
		t.Fatalf("create command request: %v", err)
	}

	response, err :=
		http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send command request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		payload, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected command status %d, got %d: %s",
			http.StatusAccepted,
			response.StatusCode,
			payload,
		)
	}
}

func waitForIntegrationState(
	t *testing.T,
	baseURL string,
	id string,
	want stream.State,
) {
	t.Helper()

	deadline := time.Now().Add(
		10 * time.Second,
	)

	for time.Now().Before(deadline) {
		response, err := http.Get(
			baseURL +
				"/api/v1/streams/" +
				id,
		)
		if err != nil {
			t.Fatalf("get stream: %v", err)
		}

		var got stream.Stream

		decodeErr := json.NewDecoder(
			response.Body,
		).Decode(&got)

		response.Body.Close()

		if decodeErr != nil {
			t.Fatalf(
				"decode stream: %v",
				decodeErr,
			)
		}

		if got.State == want {
			return
		}

		time.Sleep(
			50 * time.Millisecond,
		)
	}

	t.Fatalf(
		"stream did not reach state %q",
		want,
	)
}

func waitForSubscribers(
	t *testing.T,
	hub *stream.Hub,
	id string,
	want int,
) {
	t.Helper()

	deadline := time.Now().Add(
		time.Second,
	)

	for time.Now().Before(deadline) {
		if hub.SubscriberCount(id) == want {
			return
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	t.Fatalf(
		"expected %d subscribers, got %d",
		want,
		hub.SubscriberCount(id),
	)
}

func readMPEGTSPayload(
	t *testing.T,
	ctx context.Context,
	conn *websocket.Conn,
) []byte {
	t.Helper()

	var media []byte

	for len(media) < 188*4 {
		messageType, payload, err :=
			conn.Read(ctx)

		if err != nil {
			t.Fatalf(
				"read websocket media: %v",
				err,
			)
		}

		if messageType != websocket.MessageBinary {
			continue
		}

		media = append(
			media,
			payload...,
		)

		if containsMPEGTSSync(media) {
			return media
		}
	}

	return media
}

func containsMPEGTSSync(
	payload []byte,
) bool {
	const packetSize = 188
	const syncByte = byte(0x47)

	for i := 0; i+packetSize < len(payload); i++ {
		if payload[i] == syncByte &&
			payload[i+packetSize] == syncByte {
			return true
		}
	}

	return false
}
