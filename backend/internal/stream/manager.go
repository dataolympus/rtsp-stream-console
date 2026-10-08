package stream

import "context"

type Runner interface {
	Start(
		ctx context.Context,
		sourceURL string,
	) (<-chan error, error)
}

type Manager struct {
	ctx      context.Context
	registry Registry
	runner   Runner
}

func NewManager(
	ctx context.Context,
	registry Registry,
	runner Runner,
) *Manager {
	return &Manager{
		ctx:      ctx,
		registry: registry,
		runner:   runner,
	}
}

func (m *Manager) Start(id string) error {
	item, err := m.registry.Get(id)
	if err != nil {
		return err
	}

	item.State = StateConnecting

	if err := m.registry.Update(item); err != nil {
		return err
	}

	done, err := m.runner.Start(
		m.ctx,
		item.URL,
	)
	if err != nil {
		item.State = StateError

		if updateErr := m.registry.Update(item); updateErr != nil {
			return updateErr
		}

		return err
	}

	item.State = StateLive

	if err := m.registry.Update(item); err != nil {
		return err
	}

	go m.watchRuntime(item.ID, done)

	return nil
}

func (m *Manager) watchRuntime(
	id string,
	done <-chan error,
) {
	runtimeErr := <-done

	item, err := m.registry.Get(id)
	if err != nil {
		return
	}

	if runtimeErr != nil {
		item.State = StateError
	} else {
		item.State = StateStopped
	}

	_ = m.registry.Update(item)
}
