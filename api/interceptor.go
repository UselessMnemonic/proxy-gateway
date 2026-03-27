package api

import (
	"context"
	"net"
)

// ShouldActivateFunc is invoked by interceptors to request target activation.
type ShouldActivateFunc func()

// Interceptor handles early traffic inspection and activation signaling.
type Interceptor interface {
	// HandleTCP processes a TCP connection before backend forwarding is enabled.
	HandleTCP(context.Context, *net.TCPConn, map[string]any, ShouldActivateFunc) error
	// HandleUDP processes UDP traffic before backend forwarding is enabled.
	HandleUDP(context.Context, *net.UDPConn, map[string]any, ShouldActivateFunc) error
}
