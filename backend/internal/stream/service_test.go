package stream

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceCreate(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)

	created, err := service.Create(
		"Camera 1",
		"rtsp://localhost:8554/camera-1",
	)
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	if created.ID == "" {
		t.Fatal("expected stream ID to be generated")
	}

	if err := uuid.Validate(created.ID); err != nil {
		t.Fatalf("expected valid UUID, got %q", created.ID)
	}

	if created.Name != "Camera 1" {
		t.Fatalf("expected name %q, got %q", "Camera 1", created.Name)
	}

	if created.URL != "rtsp://localhost:8554/camera-1" {
		t.Fatalf(
			"expected URL %q, got %q",
			"rtsp://localhost:8554/camera-1",
			created.URL,
		)
	}

	if created.State != StateCreated {
		t.Fatalf(
			"expected state %q, got %q",
			StateCreated,
			created.State,
		)
	}

	if created.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if created.CreatedAt.Location() != time.UTC {
		t.Fatalf(
			"expected CreatedAt in UTC, got %v",
			created.CreatedAt.Location(),
		)
	}

	stored, err := registry.Get(created.ID)
	if err != nil {
		t.Fatalf("get stored stream: %v", err)
	}

	if stored != created {
		t.Fatalf(
			"expected stored stream %+v, got %+v",
			created,
			stored,
		)
	}
}

func TestServiceCreateTrimsInput(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)

	created, err := service.Create(
		"  Camera 1  ",
		"  rtsp://localhost:8554/camera-1  ",
	)
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	if created.Name != "Camera 1" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}

	if created.URL != "rtsp://localhost:8554/camera-1" {
		t.Fatalf("expected trimmed URL, got %q", created.URL)
	}
}

func TestServiceCreateRejectsMissingName(t *testing.T) {
	registry := NewMemoryRegistry()
	service := NewService(registry)

	_, err := service.Create(
		"   ",
		"rtsp://localhost:8554/camera-1",
	)

	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestServiceCreateRejectsInvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "empty URL",
			url:  "",
		},
		{
			name: "HTTP URL",
			url:  "https://example.com/video",
		},
		{
			name: "missing host",
			url:  "rtsp://",
		},
		{
			name: "plain text",
			url:  "camera-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewMemoryRegistry()
			service := NewService(registry)

			_, err := service.Create(
				"Camera 1",
				tt.url,
			)

			if !errors.Is(err, ErrInvalidURL) {
				t.Fatalf(
					"expected ErrInvalidURL for %q, got %v",
					tt.url,
					err,
				)
			}
		})
	}
}
