package pluginhosts

import (
	"context"
	"fmt"
	"log"
	"proxy-gateway/internal/v2/contracts"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Manager owns plugin-host lifecycle and routes activation hints to targets.
type Manager struct {
	mu    sync.RWMutex
	hosts map[string]contracts.PluginHost
	sink  contracts.ActivationSink
}

func NewManager(sink contracts.ActivationSink) *Manager {
	return &Manager{
		hosts: make(map[string]contracts.PluginHost),
		sink:  sink,
	}
}

func (m *Manager) Register(host contracts.PluginHost) error {
	if host == nil {
		return fmt.Errorf("plugin host is nil")
	}
	name := host.Name()
	if name == "" {
		return fmt.Errorf("plugin host name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.hosts[name]; exists {
		return fmt.Errorf("plugin host %q already registered", name)
	}
	m.hosts[name] = host
	return nil
}

func (m *Manager) Start(group *errgroup.Group, groupCtx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, host := range m.hosts {
		h := host
		group.Go(func() error {
			log.Printf("[plugin-host:%s] starting", h.Name())
			err := h.Start(groupCtx, m.sink)
			if err != nil {
				return fmt.Errorf("plugin-host %s failed: %w", h.Name(), err)
			}
			log.Printf("[plugin-host:%s] stopped", h.Name())
			return nil
		})
	}
}

// HintActivate allows Manager to be used directly as an activation sink bridge.
func (m *Manager) HintActivate(targetName string) {
	if m.sink == nil {
		return
	}
	m.sink.HintActivate(targetName)
}
