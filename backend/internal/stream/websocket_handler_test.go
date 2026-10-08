package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func waitForSubscriberCount(
	t *testing.T,
	hub *Hub,
	streamID string,
	want int,
) {
	t.Helper()

	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		if got := hub.SubscriberCount(streamID); got == want {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf(
		"expected %d subscribers, got %d",
		want,
		hub.SubscriberCount(streamID),
	)
}

func TestWebSocketHandlerStreamsHubPayload(t *testing.T) {
	hub := NewHub()
	handler := NewWebSocketHandler(hub)

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			r.SetPathValue(
				"id",
				"stream-1",
			)

			handler.Stream(
				w,
				r,
			)
		}),
	)
	defer server.Close()

	wsURL := "ws" +
		strings.TrimPrefix(
			server.URL,
			"http",
		)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

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

	waitForSubscriberCount(
		t,
		hub,
		"stream-1",
		1,
	)

	if hub.SubscriberCount("stream-1") != 1 {
		t.Fatal(
			"websocket handler did not subscribe to hub",
		)
	}

	hub.Publish(
		"stream-1",
		[]byte("media-data"),
	)

	messageType, payload, err := conn.Read(ctx)
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

	if string(payload) != "media-data" {
		t.Fatalf(
			"expected payload %q, got %q",
			"media-data",
			payload,
		)
	}
}

func TestWebSocketHandlerUnsubscribesWhenClientDisconnects(
	t *testing.T,
) {
	hub := NewHub()
	handler := NewWebSocketHandler(hub)

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			r.SetPathValue(
				"id",
				"stream-1",
			)

			handler.Stream(w, r)
		}),
	)
	defer server.Close()

	wsURL := "ws" +
		strings.TrimPrefix(
			server.URL,
			"http",
		)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

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

	waitForSubscriberCount(
		t,
		hub,
		"stream-1",
		1,
	)

	conn.CloseNow()

	waitForSubscriberCount(
		t,
		hub,
		"stream-1",
		0,
	)
}
