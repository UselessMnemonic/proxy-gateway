package main

import (
	"context"
	"errors"
	"net"
	"proxy-gateway/api"
	"time"
)

type Noop struct{}

func (Noop) HandleTCP(_ context.Context, _ *net.TCPConn, _ map[string]any, shouldActivate api.ShouldActivateFunc) error {
	shouldActivate()
	return nil
}

func (Noop) HandleUDP(_ context.Context, conn *net.UDPConn, _ map[string]any, shouldActivate api.ShouldActivateFunc) error {
	buf := make([]byte, 1)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, _, err := conn.ReadFromUDP(buf)
		if err == nil {
			shouldActivate()
			continue
		}
		if errors.Is(err, net.ErrClosed) {
			return net.ErrClosed
		}
	}
}

func (Noop) OnActivate(_ map[string]any) error {
	return nil
}

func (Noop) OnDeactivate(_ map[string]any) error {
	return nil
}

func DefaultActivators() map[string]api.Activator {
	result := make(map[string]api.Activator)
	result["noop"] = Noop{}
	return result
}

func DefaultInterceptors() map[string]api.Interceptor {
	result := make(map[string]api.Interceptor)
	result["noop"] = Noop{}
	return result
}
