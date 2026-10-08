package stream

import (
	"context"
	"errors"
	"io"
	"sync"
)

var (
	ErrAlreadyRunning = errors.New("stream is already running")
	ErrNotRunning     = errors.New("stream is not running")
)

type Runtime struct {
	Output io.ReadCloser
	Done   <-chan error
}

type Runner interface {
	Start(
		ctx context.Context,
		sourceURL string,
	) (*Runtime, error)
}

type Manager struct {
	ctx      context.Context
	registry Registry
	runner   Runner
	pump     *MediaPump

	mu     sync.Mutex
	active map[string]context.CancelFunc
}

func NewManager(
	ctx context.Context,
	registry Registry,
	runner Runner,
	pump *MediaPump,
) *Manager {
	return &Manager{
		ctx:      ctx,
		registry: registry,
		runner:   runner,
		pump:     pump,
		active:   make(map[string]context.CancelFunc),
	}
}

func (m *Manager) Start(id string) error {
	item, err := m.registry.Get(id)
	if err != nil {
		return err
	}

	streamCtx, cancel := context.WithCancel(m.ctx)

	m.mu.Lock()

	if _, exists := m.active[id]; exists {
		m.mu.Unlock()
		cancel()

		return ErrAlreadyRunning
	}

	m.active[id] = cancel

	m.mu.Unlock()

	item.State = StateConnecting

	if err := m.registry.Update(item); err != nil {
		m.removeActive(id)
		cancel()

		return err
	}

	runtime, err := m.runner.Start(
		streamCtx,
		item.URL,
	)
	if err != nil {
		wasCancelled := streamCtx.Err() != nil

		m.removeActive(id)
		cancel()

		if wasCancelled {
			item.State = StateStopped
		} else {
			item.State = StateError
		}

		if updateErr := m.registry.Update(item); updateErr != nil {
			return updateErr
		}

		return err
	}

	go m.pumpRuntime(
		id,
		streamCtx,
		runtime.Output,
	)

	// Stop may have been requested while Runner.Start was still working.
	if streamCtx.Err() != nil {
		go m.watchRuntime(
			id,
			streamCtx,
			runtime.Done,
		)
		return nil
	}

	item.State = StateLive

	if err := m.registry.Update(item); err != nil {
		m.removeActive(id)
		cancel()

		return err
	}

	go m.watchRuntime(
		id,
		streamCtx,
		runtime.Done,
	)

	return nil
}

func (m *Manager) Stop(id string) error {
	item, err := m.registry.Get(id)
	if err != nil {
		return err
	}

	m.mu.Lock()
	cancel, exists := m.active[id]
	m.mu.Unlock()

	if !exists {
		return ErrNotRunning
	}

	item.State = StateStopping

	if err := m.registry.Update(item); err != nil {
		return err
	}

	cancel()

	return nil
}

func (m *Manager) watchRuntime(
	id string,
	streamCtx context.Context,
	done <-chan error,
) {
	runtimeErr := <-done

	wasCancelled := streamCtx.Err() != nil

	cancel := m.removeActive(id)
	if cancel != nil {
		cancel()
	}

	item, err := m.registry.Get(id)
	if err != nil {
		return
	}

	switch {
	case item.State == StateError:
		// Preserve the failure state set by another
		// part of the runtime pipeline.

	case item.State == StateStopping:
		item.State = StateStopped

	case wasCancelled:
		item.State = StateStopped

	case runtimeErr != nil:
		item.State = StateError

	default:
		item.State = StateStopped
	}

	_ = m.registry.Update(item)
}

func (m *Manager) removeActive(
	id string,
) context.CancelFunc {
	m.mu.Lock()
	defer m.mu.Unlock()

	cancel := m.active[id]

	delete(m.active, id)

	return cancel
}

func (m *Manager) pumpRuntime(
	id string,
	streamCtx context.Context,
	output io.ReadCloser,
) {
	defer output.Close()

	err := m.pump.Run(
		id,
		output,
	)

	if err == nil {
		return
	}

	// If the stream was deliberately cancelled,
	// a media read failure during shutdown is expected.
	if streamCtx.Err() != nil {
		return
	}

	item, getErr := m.registry.Get(id)
	if getErr != nil {
		return
	}

	if item.State == StateStopping {
		return
	}

	item.State = StateError

	if updateErr := m.registry.Update(item); updateErr != nil {
		return
	}

	m.cancelActive(id)
}

func (m *Manager) cancelActive(
	id string,
) {
	m.mu.Lock()
	cancel := m.active[id]
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}
