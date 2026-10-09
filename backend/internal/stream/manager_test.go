package stream

import (
	"context"
	"errors"
	"io"
	"strings"
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

	output io.ReadCloser
}

type failingReadCloser struct{}

func (failingReadCloser) Read(
	[]byte,
) (int, error) {
	return 0, errors.New(
		"media read failed",
	)
}

func (failingReadCloser) Close() error {
	return nil
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		started:      make(chan struct{}),
		releaseStart: make(chan struct{}),
		done:         make(chan error, 1),
		cancelled:    make(chan struct{}),
	}
}

func resetFakeRunnerForNextStart(
	r *fakeRunner,
) {
	r.started = make(chan struct{})
	r.releaseStart = make(chan struct{})
	r.done = make(chan error, 1)
	r.cancelled = make(chan struct{})
	r.startErr = nil
	r.output = nil
}

func newTestManager(
	ctx context.Context,
	registry Registry,
	runner Runner,
) *Manager {
	hub := NewHub()
	pump := NewMediaPump(hub)

	return NewManager(
		ctx,
		registry,
		runner,
		pump,
	)
}

func (r *fakeRunner) Start(
	ctx context.Context,
	sourceURL string,
) (*Runtime, error) {
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

	output := r.output

	if output == nil {
		output = io.NopCloser(
			strings.NewReader("test-media"),
		)
	}

	return &Runtime{
		Output: output,
		Done:   r.done,
	}, nil
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

func TestManagerTransitionsStreamToLiveAfterMediaArrives(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf(
			"create stream: %v",
			err,
		)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	manager := newTestManager(
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
		t.Fatalf(
			"get connecting stream: %v",
			err,
		)
	}

	if connecting.State != StateConnecting {
		t.Fatalf(
			"expected state %q while runtime starts, got %q",
			StateConnecting,
			connecting.State,
		)
	}

	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf(
			"start stream: %v",
			err,
		)
	}

	live := waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	if live.State != StateLive {
		t.Fatalf(
			"expected state %q after media arrives, got %q",
			StateLive,
			live.State,
		)
	}

	if runner.sourceURL != item.URL {
		t.Fatalf(
			"expected runner source URL %q, got %q",
			item.URL,
			runner.sourceURL,
		)
	}

	if err := manager.Stop(item.ID); err != nil {
		t.Fatalf(
			"stop stream: %v",
			err,
		)
	}

	<-runner.cancelled

	runner.done <- errors.New(
		"process terminated",
	)

	waitForState(
		t,
		registry,
		item.ID,
		StateStopped,
	)
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

	manager := newTestManager(
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

	if got.Error != "stream processor failed to start" {
		t.Fatalf(
			"expected start failure error %q, got %q",
			"stream processor failed to start",
			got.Error,
		)
	}
}

func TestManagerStartStreamNotFound(t *testing.T) {
	registry := NewMemoryRegistry()
	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := newTestManager(
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

func TestManagerUnexpectedRuntimeCompletionTransitionsStreamToError(t *testing.T) {
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

	manager := newTestManager(
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

	got := waitForState(
		t,
		registry,
		item.ID,
		StateError,
	)

	if got.Error != "stream source became unavailable" {
		t.Fatalf(
			"expected runtime completion error %q, got %q",
			"stream source became unavailable",
			got.Error,
		)
	}
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

	manager := newTestManager(
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

	got := waitForState(
		t,
		registry,
		item.ID,
		StateError,
	)

	if got.Error != "stream source became unavailable" {
		t.Fatalf(
			"expected error %q, got %q",
			"stream source became unavailable",
			got.Error,
		)
	}
}

func TestManagerRetryClearsPreviousError(t *testing.T) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateError,
		Error:     "stream source became unavailable",
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	manager := newTestManager(
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
		t.Fatalf(
			"get connecting stream: %v",
			err,
		)
	}

	if connecting.State != StateConnecting {
		t.Fatalf(
			"expected state %q, got %q",
			StateConnecting,
			connecting.State,
		)
	}

	if connecting.Error != "" {
		t.Fatalf(
			"expected previous error to be cleared, got %q",
			connecting.Error,
		)
	}

	close(runner.releaseStart)

	if err := <-startResult; err != nil {
		t.Fatalf(
			"retry stream: %v",
			err,
		)
	}

	live := waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	if live.Error != "" {
		t.Fatalf(
			"expected live stream error to be empty, got %q",
			live.Error,
		)
	}

	// Clean up the fake runtime.
	if err := manager.Stop(item.ID); err != nil {
		t.Fatalf(
			"stop stream: %v",
			err,
		)
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

	manager := newTestManager(
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

	manager := newTestManager(
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

	manager := newTestManager(
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

	manager := newTestManager(
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

	manager := newTestManager(
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

func TestManagerPublishesRuntimeOutputToHub(t *testing.T) {
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

	hub := NewHub()

	media, unsubscribe := hub.Subscribe(item.ID)
	defer unsubscribe()

	pump := NewMediaPump(hub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(
		ctx,
		registry,
		runner,
		pump,
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

	select {
	case got := <-media:
		if string(got) != "test-media" {
			t.Fatalf(
				"expected media %q, got %q",
				"test-media",
				got,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for runtime media")
	}

	if err := manager.Stop(item.ID); err != nil {
		t.Fatalf("stop stream: %v", err)
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

func TestManagerPumpFailureTransitionsStreamToError(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf(
			"create stream: %v",
			err,
		)
	}

	runner := newFakeRunner()
	runner.output = failingReadCloser{}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	manager := newTestManager(
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
		t.Fatalf(
			"start stream: %v",
			err,
		)
	}

	got := waitForState(
		t,
		registry,
		item.ID,
		StateError,
	)

	if got.Error != "media stream failed" {
		t.Fatalf(
			"expected media failure error %q, got %q",
			"media stream failed",
			got.Error,
		)
	}

	select {
	case <-runner.cancelled:

	case <-time.After(time.Second):
		t.Fatal(
			"runtime was not cancelled after media failure",
		)
	}

	// Allow watchRuntime to finish cleanly.
	runner.done <- errors.New(
		"process terminated",
	)

	final := waitForState(
		t,
		registry,
		item.ID,
		StateError,
	)

	if final.Error != "media stream failed" {
		t.Fatalf(
			"expected media failure to be preserved, got %q",
			final.Error,
		)
	}
}

func TestManagerRemainsConnectingUntilMediaArrives(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	item := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://localhost:8554/camera-1",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(item); err != nil {
		t.Fatalf(
			"create stream: %v",
			err,
		)
	}

	reader, writer := io.Pipe()
	defer writer.Close()

	runner := newFakeRunner()
	runner.output = reader

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	manager := newTestManager(
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
		t.Fatalf(
			"start stream: %v",
			err,
		)
	}

	connecting, err := registry.Get(item.ID)
	if err != nil {
		t.Fatalf(
			"get stream: %v",
			err,
		)
	}

	if connecting.State != StateConnecting {
		t.Fatalf(
			"expected state %q before media arrives, got %q",
			StateConnecting,
			connecting.State,
		)
	}

	if _, err := writer.Write(
		[]byte("test-media"),
	); err != nil {
		t.Fatalf(
			"write media: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"close media writer: %v",
			err,
		)
	}

	waitForState(
		t,
		registry,
		item.ID,
		StateLive,
	)

	if err := manager.Stop(item.ID); err != nil {
		t.Fatalf(
			"stop stream: %v",
			err,
		)
	}

	<-runner.cancelled

	runner.done <- errors.New(
		"process terminated",
	)

	waitForState(
		t,
		registry,
		item.ID,
		StateStopped,
	)
}

func TestManagerRejectsStartWhenActiveStreamLimitReached(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	first := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://camera.example.com/one",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	second := Stream{
		ID:        "stream-2",
		Name:      "Camera 2",
		URL:       "rtsp://camera.example.com/two",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(first); err != nil {
		t.Fatalf("create first stream: %v", err)
	}

	if err := registry.Create(second); err != nil {
		t.Fatalf("create second stream: %v", err)
	}

	runner := newFakeRunner()

	ctx, cancel :=
		context.WithCancel(context.Background())
	defer cancel()

	hub := NewHub()
	pump := NewMediaPump(hub)

	manager := NewManagerWithMaxActiveStreams(
		ctx,
		registry,
		runner,
		pump,
		1,
	)

	firstStart := make(chan error, 1)

	go func() {
		firstStart <- manager.Start(first.ID)
	}()

	// The first stream has reserved its active runtime
	// slot and is now waiting inside Runner.Start.
	<-runner.started

	err := manager.Start(second.ID)

	if !errors.Is(
		err,
		ErrActiveStreamLimitReached,
	) {
		t.Fatalf(
			"expected ErrActiveStreamLimitReached, got %v",
			err,
		)
	}

	gotSecond, getErr :=
		registry.Get(second.ID)

	if getErr != nil {
		t.Fatalf(
			"get second stream: %v",
			getErr,
		)
	}

	if gotSecond.State != StateCreated {
		t.Fatalf(
			"expected rejected stream to remain %q, got %q",
			StateCreated,
			gotSecond.State,
		)
	}

	close(runner.releaseStart)

	if err := <-firstStart; err != nil {
		t.Fatalf(
			"start first stream: %v",
			err,
		)
	}

	if err := manager.Stop(first.ID); err != nil {
		t.Fatalf(
			"stop first stream: %v",
			err,
		)
	}

	runner.done <- context.Canceled

	waitForState(
		t,
		registry,
		first.ID,
		StateStopped,
	)
}

func TestManagerReleasesActiveStreamSlotAfterStop(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	first := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://camera.example.com/one",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	second := Stream{
		ID:        "stream-2",
		Name:      "Camera 2",
		URL:       "rtsp://camera.example.com/two",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(first); err != nil {
		t.Fatalf(
			"create first stream: %v",
			err,
		)
	}

	if err := registry.Create(second); err != nil {
		t.Fatalf(
			"create second stream: %v",
			err,
		)
	}

	runner := newFakeRunner()

	ctx, cancel :=
		context.WithCancel(context.Background())
	defer cancel()

	hub := NewHub()
	pump := NewMediaPump(hub)

	manager := NewManagerWithMaxActiveStreams(
		ctx,
		registry,
		runner,
		pump,
		1,
	)

	firstStart := make(chan error, 1)

	go func() {
		firstStart <- manager.Start(first.ID)
	}()

	<-runner.started

	close(runner.releaseStart)

	if err := <-firstStart; err != nil {
		t.Fatalf(
			"start first stream: %v",
			err,
		)
	}

	if err := manager.Stop(first.ID); err != nil {
		t.Fatalf(
			"stop first stream: %v",
			err,
		)
	}

	// Wait for the first runtime's context cancellation
	// before replacing the fake runner's per-run channels.
	<-runner.cancelled

	runner.done <- context.Canceled

	waitForState(
		t,
		registry,
		first.ID,
		StateStopped,
	)

	resetFakeRunnerForNextStart(
		runner,
	)

	secondStart := make(chan error, 1)

	go func() {
		secondStart <- manager.Start(second.ID)
	}()

	<-runner.started

	close(runner.releaseStart)

	if err := <-secondStart; err != nil {
		t.Fatalf(
			"expected second stream to start after slot release, got %v",
			err,
		)
	}

	if err := manager.Stop(second.ID); err != nil {
		t.Fatalf(
			"stop second stream: %v",
			err,
		)
	}

	runner.done <- context.Canceled

	waitForState(
		t,
		registry,
		second.ID,
		StateStopped,
	)
}

func TestManagerReleasesActiveStreamSlotAfterRuntimeFailure(
	t *testing.T,
) {
	registry := NewMemoryRegistry()

	first := Stream{
		ID:        "stream-1",
		Name:      "Camera 1",
		URL:       "rtsp://camera.example.com/one",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	second := Stream{
		ID:        "stream-2",
		Name:      "Camera 2",
		URL:       "rtsp://camera.example.com/two",
		State:     StateCreated,
		CreatedAt: time.Now().UTC(),
	}

	if err := registry.Create(first); err != nil {
		t.Fatalf(
			"create first stream: %v",
			err,
		)
	}

	if err := registry.Create(second); err != nil {
		t.Fatalf(
			"create second stream: %v",
			err,
		)
	}

	runner := newFakeRunner()

	ctx, cancel :=
		context.WithCancel(context.Background())
	defer cancel()

	hub := NewHub()
	pump := NewMediaPump(hub)

	manager := NewManagerWithMaxActiveStreams(
		ctx,
		registry,
		runner,
		pump,
		1,
	)

	firstStart := make(chan error, 1)

	go func() {
		firstStart <- manager.Start(first.ID)
	}()

	<-runner.started

	close(runner.releaseStart)

	if err := <-firstStart; err != nil {
		t.Fatalf(
			"start first stream: %v",
			err,
		)
	}

	waitForState(
		t,
		registry,
		first.ID,
		StateLive,
	)

	runner.done <- errors.New(
		"runtime failed",
	)

	waitForState(
		t,
		registry,
		first.ID,
		StateError,
	)

	// Give watchRuntime a deterministic checkpoint:
	// the failed runtime must no longer consume capacity.
	deadline := time.Now().Add(time.Second)

	for {
		manager.mu.Lock()
		activeCount := len(manager.active)
		manager.mu.Unlock()

		if activeCount == 0 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"expected failed runtime slot to be released, got %d active streams",
				activeCount,
			)
		}

		time.Sleep(10 * time.Millisecond)
	}

	resetFakeRunnerForNextStart(
		runner,
	)

	secondStart := make(chan error, 1)

	go func() {
		secondStart <- manager.Start(second.ID)
	}()

	<-runner.started

	close(runner.releaseStart)

	if err := <-secondStart; err != nil {
		t.Fatalf(
			"expected second stream to start after failure released slot, got %v",
			err,
		)
	}

	if err := manager.Stop(second.ID); err != nil {
		t.Fatalf(
			"stop second stream: %v",
			err,
		)
	}

	<-runner.cancelled

	runner.done <- context.Canceled

	waitForState(
		t,
		registry,
		second.ID,
		StateStopped,
	)
}
