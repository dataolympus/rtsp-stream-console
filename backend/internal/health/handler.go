package health

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	isReady func() bool
}

type response struct {
	Status string `json:"status"`
}

func New(isReady func() bool) *Handler {
	if isReady == nil {
		isReady = func() bool {
			return true
		}
	}

	return &Handler{
		isReady: isReady,
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{
		Status: "ok",
	})
}

func (h *Handler) Readyz(w http.ResponseWriter, _ *http.Request) {
	if !h.isReady() {
		writeJSON(w, http.StatusServiceUnavailable, response{
			Status: "not_ready",
		})
		return
	}

	writeJSON(w, http.StatusOK, response{
		Status: "ready",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(body)
}
