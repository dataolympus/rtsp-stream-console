package stream

import "time"

type State string

const (
	StateCreated    State = "created"
	StateConnecting State = "connecting"
	StateLive       State = "live"
	StateError      State = "error"
	StateStopped    State = "stopped"
)

type Stream struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	State     State     `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
}
