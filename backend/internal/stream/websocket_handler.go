package stream

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

type WebSocketHandler struct {
	hub *Hub
}

func NewWebSocketHandler(
	hub *Hub,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
	}
}

func (h *WebSocketHandler) Stream(
	w http.ResponseWriter,
	r *http.Request,
) {
	streamID := r.PathValue("id")

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
