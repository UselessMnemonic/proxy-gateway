package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"proxy-gateway/api"
	"proxy-gateway/misc"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type FrontendContext struct {
	name              string
	protocol          api.Protocol
	listen            netip.AddrPort
	target            *TargetContext
	targetService     *TargetServiceContext
	flowTimeout       api.TTL
	interceptor       api.Interceptor
	interceptorConfig map[string]any
	state             atomic.Int32
	err               misc.AtomicBox[error]
}

func (ctx *FrontendContext) Name() string {
	return ctx.name
}

func (ctx *FrontendContext) State() (api.FrontendState, error) {
	return api.FrontendState(ctx.state.Load()), ctx.err.Load()
}

func (ctx *FrontendContext) updateState(state api.FrontendState, err error) {
	ctx.state.Store(int32(state))
	ctx.err.Store(err)
}

func LoadFrontendContext(rCtx *RuntimeContext, cfg api.FrontendConfig) (*FrontendContext, error) {
	target, ok := rCtx.Targets[cfg.Forward.Target]
	if !ok {
		return nil, fmt.Errorf("target not found: %s", cfg.Forward.Target)
	}
	targetService, ok := target.services[cfg.Forward.Service]
	if !ok {
		return nil, fmt.Errorf("service  %s not found in target %s", cfg.Forward.Service, cfg.Forward.Target)
	}

	interceptor, ok := rCtx.Interceptors[cfg.Intercept.Kind]
	if !ok {
		return nil, fmt.Errorf("unknown interceptor kind %q", cfg.Intercept.Kind)
	}
	if interceptor == nil {
		return nil, fmt.Errorf("interceptor kind %q resolved to nil", cfg.Intercept.Kind)
	}

	frontendCtx := &FrontendContext{
		name:              cfg.Name,
		protocol:          cfg.Protocol,
		listen:            cfg.Listen,
		target:            target,
		targetService:     targetService,
		flowTimeout:       cfg.FlowTimeout,
		interceptor:       interceptor,
		interceptorConfig: cfg.Intercept.Config,
	}
	targetService.frontend = frontendCtx
	frontendCtx.state.Store(int32(api.FrontendStateStopped))
	return frontendCtx, nil
}

func (ctx *FrontendContext) Start(group *errgroup.Group, groupCtx context.Context) {
	groupCtx, cancel := context.WithCancel(groupCtx)
	activationHints := make(chan struct{}, 1)
	shouldActivate := api.ShouldActivateFunc(func() {
		select {
		case <-groupCtx.Done():
			return
		case activationHints <- struct{}{}:
		default:
			// Best-effort activation hint; drop when queue is full.
		}
	})

	go func() {
		for {
			select {
			case <-groupCtx.Done():
				return
			case <-activationHints:
				_ = ctx.target.Activate()
			}
		}
	}()

	switch ctx.protocol {
	case api.ProtocolTCP:
		group.Go(func() error {
			defer cancel()
			runTCP(ctx, group, groupCtx, shouldActivate)
			return nil
		})
	case api.ProtocolUDP:
		group.Go(func() error {
			defer cancel()
			runUDP(ctx, group, groupCtx, shouldActivate)
			return nil
		})
	default:
		ctx.updateState(
			api.FrontendStateStopped,
			fmt.Errorf("[frontend:%s] has unsupported protocol: %s\n", ctx.name, ctx.protocol.String()),
		)
		cancel()
	}
}

func runTCP(ctx *FrontendContext, group *errgroup.Group, groupCtx context.Context, shouldActivate api.ShouldActivateFunc) {
	listener, err := net.ListenTCP("tcp", net.TCPAddrFromAddrPort(ctx.listen))
	if err != nil {
		ctx.updateState(api.FrontendStateStopped, err)
		log.Printf("[frontend:%s] tcp bind failed: %v", ctx.name, err)
		return
	}
	ctx.updateState(api.FrontendStateRunning, nil)
	log.Printf("[frontend:%s] tcp listening on %s", ctx.name, ctx.listen)
	defer log.Printf("[frontend:%s] tcp listen stopped", ctx.name)

	go func() {
		<-groupCtx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				ctx.updateState(api.FrontendStateStopped, nil)
				return
			}
			log.Printf("[frontend:%s] tcp accept failed: %v", ctx.name, err)
			continue
		}
		group.Go(func() error {
			defer conn.Close()
			err := ctx.interceptor.HandleTCP(groupCtx, conn, ctx.interceptorConfig, shouldActivate)
			ctx.updateState(api.FrontendStateRunning, err)
			return nil
		})
	}
}

func runUDP(ctx *FrontendContext, group *errgroup.Group, groupCtx context.Context, shouldActivate api.ShouldActivateFunc) {
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(ctx.listen))
	if err != nil {
		ctx.updateState(api.FrontendStateStopped, err)
		log.Printf("[frontend:%s] udp bind failed: %v", ctx.name, err)
		return
	}
	ctx.updateState(api.FrontendStateRunning, nil)
	log.Printf("[frontend:%s] udp listening on %s", ctx.name, ctx.listen)
	defer log.Printf("[frontend:%s] udp listen stopped", ctx.name)

	go func() {
		<-groupCtx.Done()
		_ = conn.Close()
	}()

	for {
		err := ctx.interceptor.HandleUDP(groupCtx, conn, ctx.interceptorConfig, shouldActivate)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				ctx.updateState(api.FrontendStateStopped, nil)
				return
			}
			ctx.updateState(api.FrontendStateStopped, err)
		} else {
			ctx.updateState(api.FrontendStateRunning, nil)
		}
	}
}
