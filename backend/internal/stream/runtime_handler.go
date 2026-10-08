package stream

import (
	"errors"
	"net/http"
)

type RuntimeController interface {
	Start(string) error
	Stop(string) error
}

type RuntimeHandler struct {
	controller RuntimeController
}

func NewRuntimeHandler(
	controller RuntimeController,
) *RuntimeHandler {
	return &RuntimeHandler{
		controller: controller,
	}
}

func (h *RuntimeHandler) Start(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")

	err := h.controller.Start(id)
	if err == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	switch {
	case errors.Is(err, ErrNotFound):
		writeJSON(
			w,
			http.StatusNotFound,
			errorResponse{Error: "stream not found"},
		)

	case errors.Is(err, ErrAlreadyRunning):
		writeJSON(
			w,
			http.StatusConflict,
			errorResponse{Error: "stream is already running"},
		)

	default:
		writeJSON(
			w,
			http.StatusInternalServerError,
			errorResponse{Error: "internal server error"},
		)
	}
}

func (h *RuntimeHandler) Stop(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")

	err := h.controller.Stop(id)
	if err == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	switch {
	case errors.Is(err, ErrNotFound):
		writeJSON(
			w,
			http.StatusNotFound,
			errorResponse{Error: "stream not found"},
		)

	case errors.Is(err, ErrNotRunning):
		writeJSON(
			w,
			http.StatusConflict,
			errorResponse{Error: "stream is not running"},
		)

	default:
		writeJSON(
			w,
			http.StatusInternalServerError,
			errorResponse{Error: "internal server error"},
		)
	}
}
