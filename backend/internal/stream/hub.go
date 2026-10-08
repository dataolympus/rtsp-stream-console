package stream

import "sync"

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[uint64]chan []byte
	nextID      uint64
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(
			map[string]map[uint64]chan []byte,
		),
	}
}

func (h *Hub) Subscribe(
	streamID string,
) (<-chan []byte, func()) {
	ch := make(chan []byte, 1)

	h.mu.Lock()

	id := h.nextID
	h.nextID++

	if h.subscribers[streamID] == nil {
		h.subscribers[streamID] =
			make(map[uint64]chan []byte)
	}

	h.subscribers[streamID][id] = ch

	h.mu.Unlock()

	var once sync.Once

	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()

			streamSubscribers :=
				h.subscribers[streamID]

			subscriber, exists :=
				streamSubscribers[id]

			if !exists {
				return
			}

			delete(streamSubscribers, id)
			close(subscriber)

			if len(streamSubscribers) == 0 {
				delete(
					h.subscribers,
					streamID,
				)
			}
		})
	}

	return ch, unsubscribe
}

func (h *Hub) Publish(
	streamID string,
	payload []byte,
) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, subscriber := range h.subscribers[streamID] {
		select {
		case subscriber <- payload:
		default:
			// Subscriber is behind.
			// Drop this chunk rather than blocking the stream.
		}
	}
}
