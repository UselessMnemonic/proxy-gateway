package main

import (
	"context"
	"net"
	"proxy-gateway/api"
)

type Noop struct{}

func (Noop) HandleTCP(_ context.Context, _ *net.TCPConn, shouldActivate api.ShouldActivateFunc) error {
	shouldActivate()
	return nil
}

func (Noop) HandleUDP(_ context.Context, _ *net.UDPConn, shouldActivate api.ShouldActivateFunc) error {
	shouldActivate()
	return nil
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
