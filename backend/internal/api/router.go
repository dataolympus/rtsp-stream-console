package api

import (
	"net/http"
	"time"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/stream"
)

type RateLimitConfig struct {
	ExpensiveRequestsPerMinute int
	TrustProxyHeaders          bool
}

func NewRouter(
	healthHandler *health.Handler,
	streamHandler *stream.Handler,
	websocketHandler *stream.WebSocketHandler,
	runtimeHandler *stream.RuntimeHandler,
) http.Handler {
	return NewRouterWithRateLimit(
		healthHandler,
		streamHandler,
		websocketHandler,
		runtimeHandler,
		RateLimitConfig{},
	)
}

func NewRouterWithRateLimit(
	healthHandler *health.Handler,
	streamHandler *stream.Handler,
	websocketHandler *stream.WebSocketHandler,
	runtimeHandler *stream.RuntimeHandler,
	rateLimitConfig RateLimitConfig,
) http.Handler {
	mux := http.NewServeMux()

	expensiveLimiter :=
		newFixedWindowLimiter(
			rateLimitConfig.
				ExpensiveRequestsPerMinute,
			time.Minute,
		)

	expensive := func(
		handler http.HandlerFunc,
	) http.Handler {
		return rateLimit(
			expensiveLimiter,
			rateLimitConfig.
				TrustProxyHeaders,
			handler,
		)
	}

	mux.HandleFunc(
		"GET /healthz",
		healthHandler.Healthz,
	)

	mux.HandleFunc(
		"GET /readyz",
		healthHandler.Readyz,
	)

	mux.Handle(
		"POST /api/v1/streams",
		expensive(
			streamHandler.Create,
		),
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

	mux.Handle(
		"POST /api/v1/streams/{id}/start",
		expensive(
			runtimeHandler.Start,
		),
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
