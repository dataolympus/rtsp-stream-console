package stream

import (
	"context"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

type WebSocketHandler struct {
	hub                 *Hub
	maxViewersPerStream int

	mu      sync.Mutex
	viewers map[string]int
}

func NewWebSocketHandler(
	hub *Hub,
) *WebSocketHandler {
	return NewWebSocketHandlerWithMaxViewers(
		hub,
		0,
	)
}

func NewWebSocketHandlerWithMaxViewers(
	hub *Hub,
	maxViewersPerStream int,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                 hub,
		maxViewersPerStream: maxViewersPerStream,
		viewers:             make(map[string]int),
	}
}

func (h *WebSocketHandler) acquireViewer(
	streamID string,
) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.maxViewersPerStream > 0 &&
		h.viewers[streamID] >=
			h.maxViewersPerStream {
		return false
	}

	h.viewers[streamID]++

	return true
}

func (h *WebSocketHandler) releaseViewer(
	streamID string,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.viewers[streamID]--

	if h.viewers[streamID] <= 0 {
		delete(
			h.viewers,
			streamID,
		)
	}
}

func (h *WebSocketHandler) Stream(
	w http.ResponseWriter,
	r *http.Request,
) {
	streamID := r.PathValue("id")

	if !h.acquireViewer(streamID) {
		http.Error(
			w,
			"viewer capacity reached",
			http.StatusServiceUnavailable,
		)

		return
	}

	defer h.releaseViewer(streamID)

	conn, err := websocket.Accept(
		w,
		r,
		nil,
	)
	if err != nil {
		return
	}
	defer conn.CloseNow()

	media, unsubscribe :=
		h.hub.Subscribe(streamID)

	defer unsubscribe()

	connCtx := conn.CloseRead(
		context.Background(),
	)

	for {
		select {
		case <-connCtx.Done():
			return

		case payload, ok := <-media:
			if !ok {
				_ = conn.Close(
					websocket.StatusNormalClosure,
					"stream closed",
				)

				return
			}

			if err := conn.Write(
				connCtx,
				websocket.MessageBinary,
				payload,
			); err != nil {
				return
			}
		}
	}
}
