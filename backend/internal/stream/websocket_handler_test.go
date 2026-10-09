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

func waitForViewerCount(
	t *testing.T,
	handler *WebSocketHandler,
	streamID string,
	want int,
) {
	t.Helper()

	deadline :=
		time.Now().Add(time.Second)

	for {
		handler.mu.Lock()
		got := handler.viewers[streamID]
		handler.mu.Unlock()

		if got == want {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for %d viewers, got %d",
				want,
				got,
			)
		}

		time.Sleep(10 * time.Millisecond)
	}
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

func TestWebSocketHandlerRejectsCrossOriginConnection(
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

	_, response, err := websocket.Dial(
		ctx,
		wsURL,
		&websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": {
					"https://evil.example",
				},
			},
		},
	)

	if err == nil {
		t.Fatal(
			"expected cross-origin websocket connection to be rejected",
		)
	}

	if response == nil {
		t.Fatal(
			"expected websocket handshake response",
		)
	}

	if response.StatusCode != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			response.StatusCode,
		)
	}

	if got := hub.SubscriberCount(
		"stream-1",
	); got != 0 {
		t.Fatalf(
			"expected no subscriber after rejected connection, got %d",
			got,
		)
	}
}

func TestWebSocketHandlerRejectsViewerWhenPerStreamLimitReached(
	t *testing.T,
) {
	hub := NewHub()

	handler :=
		NewWebSocketHandlerWithMaxViewers(
			hub,
			1,
		)

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

	first, _, err := websocket.Dial(
		ctx,
		wsURL,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"connect first viewer: %v",
			err,
		)
	}
	defer first.CloseNow()

	waitForSubscriberCount(
		t,
		hub,
		"stream-1",
		1,
	)

	second, response, err :=
		websocket.Dial(
			ctx,
			wsURL,
			nil,
		)

	if second != nil {
		second.CloseNow()
	}

	if err == nil {
		t.Fatal(
			"expected second viewer to be rejected",
		)
	}

	if response == nil {
		t.Fatal(
			"expected HTTP response for rejected viewer",
		)
	}

	if response.StatusCode !=
		http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			response.StatusCode,
		)
	}

	if got := hub.SubscriberCount(
		"stream-1",
	); got != 1 {
		t.Fatalf(
			"expected first viewer to remain subscribed, got %d subscribers",
			got,
		)
	}
}

func TestWebSocketHandlerReleasesViewerSlotAfterDisconnect(
	t *testing.T,
) {
	hub := NewHub()

	handler :=
		NewWebSocketHandlerWithMaxViewers(
			hub,
			1,
		)

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
		2*time.Second,
	)
	defer cancel()

	first, _, err := websocket.Dial(
		ctx,
		wsURL,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"connect first viewer: %v",
			err,
		)
	}

	waitForViewerCount(
		t,
		handler,
		"stream-1",
		1,
	)

	if err := first.Close(
		websocket.StatusNormalClosure,
		"done",
	); err != nil {
		t.Fatalf(
			"close first viewer: %v",
			err,
		)
	}

	waitForViewerCount(
		t,
		handler,
		"stream-1",
		0,
	)

	second, response, err :=
		websocket.Dial(
			ctx,
			wsURL,
			nil,
		)

	if err != nil {
		status := 0

		if response != nil {
			status = response.StatusCode
		}

		t.Fatalf(
			"expected second viewer to connect after slot release, status=%d err=%v",
			status,
			err,
		)
	}
	defer second.CloseNow()

	waitForViewerCount(
		t,
		handler,
		"stream-1",
		1,
	)
}
