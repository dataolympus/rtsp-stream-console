package stream

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRunner struct {
	started      chan struct{}
	releaseStart chan struct{}
	done         chan error
	cancelled    chan struct{}
	startErr     error
	sourceURL    string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		started:      make(chan struct{}),
		releaseStart: make(chan struct{}),
		done:         make(chan error, 1),
		cancelled:    make(chan struct{}),
	}
}

func (r *fakeRunner) Start(
	ctx context.Context,
	sourceURL string,
) (<-chan error, error) {
	r.sourceURL = sourceURL

	close(r.started)

	select {
	case <-r.releaseStart:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if r.startErr != nil {
		return nil, r.startErr
	}

	go func() {
		<-ctx.Done()
		close(r.cancelled)
	}()

	return r.done, nil
}

func waitForState(
	t *testing.T,
	registry Registry,
	id string,
	want State,
) Stream {
	t.Helper()

	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		got, err := registry.Get(id)
		if err != nil {
			t.Fatalf("get stream: %v", err)
		}

		if got.State == want {
			return got
		}

		time.Sleep(10 * time.Millisecond)
	}

	got, err := registry.Get(id)
	if err != nil {
		t.Fatalf("get stream after timeout: %v", err)
	}

	t.Fatalf(
		"expected state %q, got %q",
		want,
		got.State,
	)

	return Stream{}
}

func TestManagerStartTransitionsStreamToLive(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started

	connecting, err := registry.Get(item.ID)
	if err != nil {
		t.Fatalf("get connecting stream: %v", err)
	}

	if connecting.State != StateConnecting {
		t.Fatalf(
			"expected state %q, got %q",
			StateConnecting,
			connecting.State,
		)
	}

	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf("start stream: %v", err)
	}

	live, err := registry.Get(item.ID)
	if err != nil {
		t.Fatalf("get live stream: %v", err)
	}

	if live.State != StateLive {
		t.Fatalf(
			"expected state %q, got %q",
			StateLive,
			live.State,
		)
	}

	if runner.sourceURL != item.URL {
		t.Fatalf(
			"expected runner URL %q, got %q",
			item.URL,
			runner.sourceURL,
		)
	}
}

func TestManagerStartFailureTransitionsStreamToError(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()
	runner.startErr = errors.New("runner failed")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started

	close(runner.releaseStart)

	err := <-startResult
	if err == nil {
		t.Fatal("expected start error")
	}

	got, getErr := registry.Get(item.ID)
	if getErr != nil {
		t.Fatalf("get stream: %v", getErr)
	}

	if got.State != StateError {
		t.Fatalf(
			"expected state %q, got %q",
			StateError,
			got.State,
		)
	}
}

func TestManagerStartStreamNotFound(t *testing.T) {
	registry := NewMemoryRegistry()
	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	err := manager.Start("missing")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf(
			"expected ErrNotFound, got %v",
			err,
		)
	}
}

func TestManagerRuntimeCompletionTransitionsStreamToStopped(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started
	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf("start stream: %v", err)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	runner.done <- nil

	waitForState(
		t,
		registry,
		item.ID,
		StateStopped,
	)
}

func TestManagerRuntimeFailureTransitionsStreamToError(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started
	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf("start stream: %v", err)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	runner.done <- errors.New("runtime failed")

	waitForState(
		t,
		registry,
		item.ID,
		StateError,
	)
}

func TestManagerStopTransitionsStreamToStopped(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started
	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf("start stream: %v", err)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	if err := manager.Stop(item.ID); err != nil {
		t.Fatalf("stop stream: %v", err)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateStopping,
	)

	select {
	case <-runner.cancelled:
	case <-time.After(time.Second):
		t.Fatal("runner context was not cancelled")
	}

	// Simulating what a real process might report after being terminated.
	runner.done <- errors.New("process terminated")

	waitForState(
		t,
		registry,
		item.ID,
		StateStopped,
	)
}

func TestManagerStopNotRunning(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	err := manager.Stop(item.ID)

	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf(
			"expected ErrNotRunning, got %v",
			err,
		)
	}
}

func TestManagerStopNotFound(t *testing.T) {
	registry := NewMemoryRegistry()
	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	err := manager.Stop("missing")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf(
			"expected ErrNotFound, got %v",
			err,
		)
	}
}

func TestManagerRejectsDuplicateStart(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started
	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf("first start: %v", err)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	err := manager.Start(item.ID)

	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf(
			"expected ErrAlreadyRunning, got %v",
			err,
		)
	}

	if err := manager.Stop(item.ID); err != nil {
		t.Fatalf("cleanup stop: %v", err)
	}

	<-runner.cancelled
	runner.done <- nil

	waitForState(
		t,
		registry,
		item.ID,
		StateStopped,
	)
}

func TestManagerApplicationCancellationStopsActiveStream(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())

	manager := NewManager(
		ctx,
		registry,
		runner,
	)

	startResult := make(chan error, 1)

	go func() {
		startResult <- manager.Start(item.ID)
	}()

	<-runner.started
	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf("start stream: %v", err)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	cancel()

	select {
	case <-runner.cancelled:
	case <-time.After(time.Second):
		t.Fatal("runner context was not cancelled")
	}

	// Simulate the real process exiting because its context was cancelled.
	runner.done <- errors.New("process terminated")

	waitForState(
		t,
		registry,
		item.ID,
		StateStopped,
	)
}
