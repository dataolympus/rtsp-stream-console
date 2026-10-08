package stream

import (
	"bytes"
	"testing"
	"time"
)

func TestHubPublishesToAllSubscribers(t *testing.T) {
	hub := NewHub()

	first, unsubscribeFirst := hub.Subscribe("stream-1")
	defer unsubscribeFirst()

	second, unsubscribeSecond := hub.Subscribe("stream-1")
	defer unsubscribeSecond()

	payload := []byte("mpeg-ts-data")

	hub.Publish(
		"stream-1",
		payload,
	)

	assertPayload := func(
		name string,
		ch <-chan []byte,
	) {
		t.Helper()

		select {
		case got := <-ch:
			if !bytes.Equal(got, payload) {
				t.Fatalf(
					"%s: expected %q, got %q",
					name,
					payload,
					got,
				)
			}

		case <-time.After(time.Second):
			t.Fatalf(
				"%s: timed out waiting for payload",
				name,
			)
		}
	}

	assertPayload(
		"first subscriber",
		first,
	)

	assertPayload(
		"second subscriber",
		second,
	)
}

func TestHubKeepsStreamsIndependent(t *testing.T) {
	hub := NewHub()

	first, unsubscribeFirst := hub.Subscribe("stream-1")
	defer unsubscribeFirst()

	second, unsubscribeSecond := hub.Subscribe("stream-2")
	defer unsubscribeSecond()

	hub.Publish(
		"stream-1",
		[]byte("camera-1-data"),
	)

	select {
	case got := <-first:
		if !bytes.Equal(
			got,
			[]byte("camera-1-data"),
		) {
			t.Fatalf(
				"unexpected first stream payload: %q",
				got,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"timed out waiting for stream-1 payload",
		)
	}

	select {
	case got := <-second:
		t.Fatalf(
			"stream-2 unexpectedly received %q",
			got,
		)

	case <-time.After(50 * time.Millisecond):
		// Expected: stream-2 receives nothing.
	}
}

func TestHubSlowSubscriberDoesNotBlockPublish(t *testing.T) {
	hub := NewHub()

	slow, unsubscribeSlow := hub.Subscribe("stream-1")
	defer unsubscribeSlow()

	fast, unsubscribeFast := hub.Subscribe("stream-1")
	defer unsubscribeFast()

	// Fill both one-slot subscriber buffers.
	hub.Publish(
		"stream-1",
		[]byte("first"),
	)

	// Drain only the fast subscriber.
	<-fast

	published := make(chan struct{})

	go func() {
		hub.Publish(
			"stream-1",
			[]byte("second"),
		)

		close(published)
	}()

	select {
	case <-published:
		// Publish must not be held hostage
		// by the slow subscriber.

	case <-time.After(100 * time.Millisecond):
		t.Fatal(
			"slow subscriber blocked publish",
		)
	}

	select {
	case got := <-fast:
		if !bytes.Equal(
			got,
			[]byte("second"),
		) {
			t.Fatalf(
				"expected fast subscriber payload %q, got %q",
				"second",
				got,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"fast subscriber did not receive second payload",
		)
	}

	_ = slow
}

func TestHubUnsubscribeRemovesSubscriber(t *testing.T) {
	hub := NewHub()

	removed, unsubscribeRemoved :=
		hub.Subscribe("stream-1")

	active, unsubscribeActive :=
		hub.Subscribe("stream-1")

	defer unsubscribeActive()

	unsubscribeRemoved()

	hub.Publish(
		"stream-1",
		[]byte("media-data"),
	)

	select {
	case _, ok := <-removed:
		if ok {
			t.Fatal(
				"unsubscribed subscriber unexpectedly received data",
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"unsubscribed subscriber channel was not closed",
		)
	}

	select {
	case got := <-active:
		if !bytes.Equal(
			got,
			[]byte("media-data"),
		) {
			t.Fatalf(
				"expected active subscriber payload %q, got %q",
				"media-data",
				got,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"active subscriber did not receive payload",
		)
	}
}

func TestHubUnsubscribeIsIdempotent(t *testing.T) {
	hub := NewHub()

	_, unsubscribe :=
		hub.Subscribe("stream-1")

	unsubscribe()
	unsubscribe()
}

func TestHubSubscriberCount(t *testing.T) {
	hub := NewHub()

	_, unsubscribeFirst :=
		hub.Subscribe("stream-1")

	_, unsubscribeSecond :=
		hub.Subscribe("stream-1")

	if got := hub.SubscriberCount("stream-1"); got != 2 {
		t.Fatalf(
			"expected 2 subscribers, got %d",
			got,
		)
	}

	unsubscribeFirst()

	if got := hub.SubscriberCount("stream-1"); got != 1 {
		t.Fatalf(
			"expected 1 subscriber, got %d",
			got,
		)
	}

	unsubscribeSecond()

	if got := hub.SubscriberCount("stream-1"); got != 0 {
		t.Fatalf(
			"expected 0 subscribers, got %d",
			got,
		)
	}
}
