package stream

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRuntimeController struct {
	startID  string
	stopID   string
	startErr error
	stopErr  error
}

func (f *fakeRuntimeController) Start(id string) error {
	f.startID = id
	return f.startErr
}

func (f *fakeRuntimeController) Stop(id string) error {
	f.stopID = id
	return f.stopErr
}

func TestRuntimeHandlerStartAccepted(t *testing.T) {
	controller := &fakeRuntimeController{}
	handler := NewRuntimeHandler(controller)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/stream-1/start",
		nil,
	)
	request.SetPathValue("id", "stream-1")

	response := httptest.NewRecorder()

	handler.Start(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			response.Code,
		)
	}

	if controller.startID != "stream-1" {
		t.Fatalf(
			"expected stream ID %q, got %q",
			"stream-1",
			controller.startID,
		)
	}
}

func TestRuntimeHandlerStartNotFound(t *testing.T) {
	controller := &fakeRuntimeController{
		startErr: ErrNotFound,
	}
	handler := NewRuntimeHandler(controller)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/missing/start",
		nil,
	)
	request.SetPathValue("id", "missing")

	response := httptest.NewRecorder()

	handler.Start(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.Code,
		)
	}
}

func TestRuntimeHandlerStartAlreadyRunning(t *testing.T) {
	controller := &fakeRuntimeController{
		startErr: ErrAlreadyRunning,
	}
	handler := NewRuntimeHandler(controller)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/stream-1/start",
		nil,
	)
	request.SetPathValue("id", "stream-1")

	response := httptest.NewRecorder()

	handler.Start(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			response.Code,
		)
	}
}

func TestRuntimeHandlerStopAccepted(t *testing.T) {
	controller := &fakeRuntimeController{}
	handler := NewRuntimeHandler(controller)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/stream-1/stop",
		nil,
	)
	request.SetPathValue("id", "stream-1")

	response := httptest.NewRecorder()

	handler.Stop(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusAccepted,
			response.Code,
		)
	}

	if controller.stopID != "stream-1" {
		t.Fatalf(
			"expected stream ID %q, got %q",
			"stream-1",
			controller.stopID,
		)
	}
}

func TestRuntimeHandlerStopNotRunning(t *testing.T) {
	controller := &fakeRuntimeController{
		stopErr: ErrNotRunning,
	}
	handler := NewRuntimeHandler(controller)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/stream-1/stop",
		nil,
	)
	request.SetPathValue("id", "stream-1")

	response := httptest.NewRecorder()

	handler.Stop(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			response.Code,
		)
	}
}

func TestRuntimeHandlerStopNotFound(t *testing.T) {
	controller := &fakeRuntimeController{
		stopErr: ErrNotFound,
	}
	handler := NewRuntimeHandler(controller)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams/missing/stop",
		nil,
	)
	request.SetPathValue("id", "missing")

	response := httptest.NewRecorder()

	handler.Stop(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.Code,
		)
	}
}
