package api

import (
	"net/http"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/stream"
)

func NewRouter(
	healthHandler *health.Handler,
	streamHandler *stream.Handler,
	websocketHandler *stream.WebSocketHandler,
	runtimeHandler *stream.RuntimeHandler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /healthz",
		healthHandler.Healthz,
	)

	mux.HandleFunc(
		"GET /readyz",
		healthHandler.Readyz,
	)

	mux.HandleFunc(
		"POST /api/v1/streams",
		streamHandler.Create,
	)

	mux.HandleFunc(
		"GET /api/v1/streams",
		streamHandler.List,
	)

	mux.HandleFunc(
		"GET /api/v1/streams/{id}",
		streamHandler.Get,
	)

	mux.HandleFunc(
		"DELETE /api/v1/streams/{id}",
		streamHandler.Delete,
	)

	mux.HandleFunc(
		"POST /api/v1/streams/{id}/start",
		runtimeHandler.Start,
	)

	mux.HandleFunc(
		"POST /api/v1/streams/{id}/stop",
		runtimeHandler.Stop,
	)

	mux.HandleFunc(
		"GET /api/v1/streams/{id}/ws",
		websocketHandler.Stream,
	)

	return mux
}
