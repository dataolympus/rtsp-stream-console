package stream

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryRegistryCreateAndGet(t *testing.T) {
	registry := NewMemoryRegistry()

	want := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
	}

	if err := registry.Create(want); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	got, err := registry.Get(want.ID)
	if err != nil {
		t.Fatalf("get stream: %v", err)
	}

	if got != want {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}

func TestMemoryRegistryGetNotFound(t *testing.T) {
	registry := NewMemoryRegistry()

	_, err := registry.Get("missing")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryRegistryRejectsDuplicateID(t *testing.T) {
	registry := NewMemoryRegistry()

	stream := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(stream); err != nil {
		t.Fatalf("first create: %v", err)
	}

	err := registry.Create(stream)

	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestMemoryRegistryListIsDeterministic(t *testing.T) {
	registry := NewMemoryRegistry()

	later := Stream{
		ID:        "stream-2",
		Name:      "Camera 2",
		URL:       "rtsp://localhost:8554/camera-2",
		State:     StateCreated,
		CreatedAt: time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC),
	}

	earlier := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
	}

	if err := registry.Create(later); err != nil {
		t.Fatalf("create later stream: %v", err)
	}

	if err := registry.Create(earlier); err != nil {
		t.Fatalf("create earlier stream: %v", err)
	}

	got := registry.List()

	if len(got) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(got))
	}

	if got[0].ID != "stream-1" {
		t.Fatalf("expected stream-1 first, got %s", got[0].ID)
	}

	if got[1].ID != "stream-2" {
		t.Fatalf("expected stream-2 second, got %s", got[1].ID)
	}
}

func TestMemoryRegistryDelete(t *testing.T) {
	registry := NewMemoryRegistry()

	stream := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(stream); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	if err := registry.Delete(stream.ID); err != nil {
		t.Fatalf("delete stream: %v", err)
	}

	_, err := registry.Get(stream.ID)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryRegistryDeleteNotFound(t *testing.T) {
	registry := NewMemoryRegistry()

	err := registry.Delete("missing")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryRegistryConcurrentCreates(t *testing.T) {
	registry := NewMemoryRegistry()

	const count = 100

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			item := Stream{
				ID:        fmt.Sprintf("stream-%d", i),
				Name:      fmt.Sprintf("Camera %d", i),
				URL:       fmt.Sprintf("rtsp://localhost/camera-%d", i),
				State:     StateCreated,
				CreatedAt: time.Now().UTC(),
			}

			if err := registry.Create(item); err != nil {
				t.Errorf("create stream %d: %v", i, err)
			}

			_ = registry.List()
		}(i)
	}

	wg.Wait()

	got := registry.List()

	if len(got) != count {
		t.Fatalf("expected %d streams, got %d", count, len(got))
	}
}
