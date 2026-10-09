package stream

import (
	"errors"
	"fmt"
	"io"
)

const mediaBufferSize = 32 * 1024

type MediaPump struct {
	hub *Hub
}

func NewMediaPump(
	hub *Hub,
) *MediaPump {
	return &MediaPump{
		hub: hub,
	}
}

func (p *MediaPump) Run(
	streamID string,
	reader io.Reader,
) error {
	buffer := make(
		[]byte,
		mediaBufferSize,
	)

	for {
		n, err := reader.Read(buffer)

		if n > 0 {
			p.hub.Publish(
				streamID,
				buffer[:n],
			)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf(
				"read media output: %w",
				err,
			)
		}
	}
}
