package stream

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxCreateStreamBodyBytes int64 = 4 * 1024

type Handler struct {
	service *Service
}

type createRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request createRequest

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxCreateStreamBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) {
			writeJSON(
				w,
				http.StatusRequestEntityTooLarge,
				errorResponse{
					Error: "request body too large",
				},
			)

			return
		}

		writeJSON(
			w,
			http.StatusBadRequest,
			errorResponse{
				Error: "invalid request body",
			},
		)

		return
	}

	created, err := h.service.CreateContext(
		r.Context(),
		request.Name,
		request.URL,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrInvalidURL),
			errors.Is(err, ErrRTSPURLNotAllowed):

			writeJSON(
				w,
				http.StatusBadRequest,
				errorResponse{
					Error: err.Error(),
				},
			)

		default:
			writeJSON(
				w,
				http.StatusInternalServerError,
				errorResponse{
					Error: "internal server error",
				},
			)
		}

		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.service.List())
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	found, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "stream not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, found)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error: "stream not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(body)
}
