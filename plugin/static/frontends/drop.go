package frontends

import (
	"net/netip"

	"github.com/UselessMnemonic/proxygw/pkg/config"
	"github.com/UselessMnemonic/proxygw/pkg/frontend"
)

// DropHandler is a frontend handler that never listens for or warms traffic.
type DropHandler struct {
	ch chan struct{}
}

// Start does not bind any listener.
func (h *DropHandler) Start() error {
	return nil
}

// Stop does not release any listener.
func (h *DropHandler) Stop() error {
	return nil
}

// Close permanently tears down the handler.
func (h *DropHandler) Close() error {
	return nil
}

// ShouldWarm never emits warm signals.
func (h *DropHandler) ShouldWarm() <-chan struct{} {
	return h.ch
}

// NewDropHandler creates a drop frontend handler.
func NewDropHandler(_ string, _ config.Protocol, _ netip.AddrPort, _ map[string]any) (frontend.Handler, error) {
	return &DropHandler{
		ch: make(chan struct{}),
	}, nil
}
