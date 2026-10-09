package stream

import "time"

type State string

const (
	StateCreated    State = "created"
	StateConnecting State = "connecting"
	StateLive       State = "live"
	StateStopping   State = "stopping"
	StateStopped    State = "stopped"
	StateError      State = "error"
)

type Stream struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	State     State     `json:"state"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
