package api

import (
	"context"
	"net"
)

type ShouldActivateFunc func()

type Interceptor interface {
	HandleTCP(context.Context, *net.TCPConn, map[string]any, ShouldActivateFunc) error
	HandleUDP(context.Context, *net.UDPConn, map[string]any, ShouldActivateFunc) error
}
