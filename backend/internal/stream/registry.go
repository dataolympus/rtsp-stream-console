package stream

import (
	"errors"
	"sort"
	"sync"
)

var (
	ErrNotFound      = errors.New("stream not found")
	ErrAlreadyExists = errors.New("stream already exists")
)

type Registry interface {
	Create(Stream) error
	Get(string) (Stream, error)
	List() []Stream
	Delete(string) error
}

type MemoryRegistry struct {
	mu      sync.RWMutex
	streams map[string]Stream
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		streams: make(map[string]Stream),
	}
}

func (r *MemoryRegistry) Create(stream Stream) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.streams[stream.ID]; exists {
		return ErrAlreadyExists
	}

	r.streams[stream.ID] = stream

	return nil
}

func (r *MemoryRegistry) Get(id string) (Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	found, exists := r.streams[id]
	if !exists {
		return Stream{}, ErrNotFound
	}

	return found, nil
}

func (r *MemoryRegistry) List() []Stream {
	r.mu.RLock()
	defer r.mu.RUnlock()

	streams := make([]Stream, 0, len(r.streams))

	for _, item := range r.streams {
		streams = append(streams, item)
	}

	sort.Slice(streams, func(i, j int) bool {
		if streams[i].CreatedAt.Equal(streams[j].CreatedAt) {
			return streams[i].ID < streams[j].ID
		}

		return streams[i].CreatedAt.Before(streams[j].CreatedAt)
	})

	return streams
}

func (r *MemoryRegistry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.streams[id]; !exists {
		return ErrNotFound
	}

	delete(r.streams, id)

	return nil
}
