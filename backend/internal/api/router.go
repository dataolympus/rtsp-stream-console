package api

import (
	"net/http"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
)

func NewRouter(healthHandler *health.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /readyz", healthHandler.Readyz)

	return mux
}
