package stream

import (
	"bytes"
	"sync"
)

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
	const subscriberBufferSize = 64

	ch := make(
		chan []byte,
		subscriberBufferSize,
	)

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
	// Publish retains payload beyond this call.
	// Clone it so callers may safely reuse their input buffer.
	ownedPayload := bytes.Clone(payload)

	h.mu.Lock()
	defer h.mu.Unlock()

	streamSubscribers :=
		h.subscribers[streamID]

	for id, subscriber := range streamSubscribers {

		select {
		case subscriber <- ownedPayload:

		default:
			// This viewer can no longer keep up with
			// the live stream. Remove it instead of
			// silently corrupting its byte stream.
			delete(
				streamSubscribers,
				id,
			)

			close(subscriber)
		}
	}

	if len(streamSubscribers) == 0 {
		delete(
			h.subscribers,
			streamID,
		)
	}
}

func (h *Hub) SubscriberCount(
	streamID string,
) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(
		h.subscribers[streamID],
	)
}
