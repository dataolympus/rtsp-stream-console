package stream

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

var errTestRead = errors.New("test read failure")

type failingReader struct{}

type chunkReader struct {
	chunks [][]byte
	index  int
}

func (r *chunkReader) Read(
	buffer []byte,
) (int, error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}

	chunk := r.chunks[r.index]
	r.index++

	return copy(buffer, chunk), nil
}

func (failingReader) Read(
	[]byte,
) (int, error) {
	return 0, errTestRead
}

func TestMediaPumpPublishesRuntimeOutput(t *testing.T) {
	hub := NewHub()

	media, unsubscribe := hub.Subscribe("stream-1")
	defer unsubscribe()

	pump := NewMediaPump(hub)

	err := pump.Run(
		"stream-1",
		strings.NewReader("media-data"),
	)
	if err != nil {
		t.Fatalf("run media pump: %v", err)
	}

	select {
	case got := <-media:
		if !bytes.Equal(
			got,
			[]byte("media-data"),
		) {
			t.Fatalf(
				"expected payload %q, got %q",
				"media-data",
				got,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for media")
	}
}

func TestMediaPumpReturnsReadError(t *testing.T) {
	hub := NewHub()
	pump := NewMediaPump(hub)

	err := pump.Run(
		"stream-1",
		failingReader{},
	)

	if !errors.Is(err, errTestRead) {
		t.Fatalf(
			"expected read error %v, got %v",
			errTestRead,
			err,
		)
	}
}

func TestMediaPumpPublishesIndependentPayloads(t *testing.T) {
	hub := NewHub()

	media, unsubscribe := hub.Subscribe("stream-1")
	defer unsubscribe()

	pump := NewMediaPump(hub)

	reader := &chunkReader{
		chunks: [][]byte{
			[]byte("first"),
			[]byte("second"),
		},
	}

	done := make(chan error, 1)

	go func() {
		done <- pump.Run(
			"stream-1",
			reader,
		)
	}()

	first := <-media

	if string(first) != "first" {
		t.Fatalf(
			"expected first payload %q, got %q",
			"first",
			first,
		)
	}

	second := <-media

	if string(second) != "second" {
		t.Fatalf(
			"expected second payload %q, got %q",
			"second",
			second,
		)
	}

	if string(first) != "first" {
		t.Fatalf(
			"first payload was mutated: %q",
			first,
		)
	}

	if err := <-done; err != nil {
		t.Fatalf("run media pump: %v", err)
	}
}
