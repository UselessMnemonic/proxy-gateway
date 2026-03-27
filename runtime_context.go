package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"proxy-gateway/api"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type RuntimeContext struct {
	Config       *api.Config
	Plugins      map[string]*PluginContext
	Targets      map[string]*TargetContext
	Frontends    map[string]*FrontendContext
	Activators   map[string]api.Activator
	Interceptors map[string]api.Interceptor
	Nftables     *NftablesContext
	Conntrack    *ConntrackContext
	Socket       *net.UnixListener
	PluginHosts  *PluginHostContext

	pluginMu      sync.RWMutex
	pluginTunnels map[string]PluginTunnel
}

func LoadRuntimeContext(cfg *api.Config) (*RuntimeContext, error) {
	var err error

	ctx := &RuntimeContext{
		Config:        cfg,
		Plugins:       make(map[string]*PluginContext),
		Targets:       make(map[string]*TargetContext, len(cfg.Targets)),
		Frontends:     make(map[string]*FrontendContext, len(cfg.Frontends)),
		Activators:    DefaultActivators(),
		Interceptors:  DefaultInterceptors(),
		pluginTunnels: make(map[string]PluginTunnel),
	}

	defer func() {
		if err != nil {
			if ctx.Socket != nil {
				_ = ctx.Socket.Close()
			}
			if ctx.Conntrack != nil && ctx.Conntrack.conn != nil {
				_ = ctx.Conntrack.conn.Close()
			}
		}
	}()

	socketAddr, err := net.ResolveUnixAddr("unix", cfg.Runtime.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve unix socket address: %v", err)
	}
	ctx.Socket, err = net.ListenUnix("unix", socketAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind unix socket: %v", err)
	}
	ctx.Socket.SetUnlinkOnClose(true)

	for _, targetCfg := range cfg.Targets {
		targetCtx, err := LoadTargetContext(ctx, targetCfg)
		if err != nil {
			return nil, fmt.Errorf("error loading target %q: %w", targetCfg.Name, err)
		}
		ctx.Targets[targetCfg.Name] = targetCtx
	}

	for _, frontendCfg := range cfg.Frontends {
		frontendCtx, err := LoadFrontendContext(ctx, frontendCfg)
		if err != nil {
			return nil, fmt.Errorf("error loading frontend %q: %w", frontendCfg.Name, err)
		}
		ctx.Frontends[frontendCfg.Name] = frontendCtx
	}

	ctx.Nftables, err = DialNftables()
	if err != nil {
		return nil, err
	}
	for fName, f := range ctx.Frontends {
		if err = ctx.Nftables.SetTTL(fName, f.protocol, f.listen, f.flowTimeout); err != nil {
			return nil, err
		}
	}

	ctx.Conntrack, err = DialConntrack(ctx)
	if err != nil {
		return nil, err
	}

	ctx.PluginHosts, err = LoadPluginHostContext(cfg)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

func (ctx *RuntimeContext) RegisterActivator(kind string, activator api.Activator) error {
	if kind == "" {
		return fmt.Errorf("activator kind is required")
	}
	if activator == nil {
		return fmt.Errorf("activator %q is nil", kind)
	}
	if _, exists := ctx.Activators[kind]; exists {
		return fmt.Errorf("duplicate activator kind %q", kind)
	}
	ctx.Activators[kind] = activator
	return nil
}

func (ctx *RuntimeContext) RegisterInterceptor(kind string, interceptor api.Interceptor) error {
	if kind == "" {
		return fmt.Errorf("interceptor kind is required")
	}
	if interceptor == nil {
		return fmt.Errorf("interceptor %q is nil", kind)
	}
	if _, exists := ctx.Interceptors[kind]; exists {
		return fmt.Errorf("duplicate interceptor kind %q", kind)
	}
	ctx.Interceptors[kind] = interceptor
	return nil
}

func (ctx *RuntimeContext) Start(group *errgroup.Group, groupCtx context.Context) {
	group.Go(func() error {
		log.Printf("[listener] started socket=%s", ctx.Socket.Addr())
		socketContext, cancel := context.WithCancel(groupCtx)
		defer cancel()
		go func() {
			<-socketContext.Done()
			_ = ctx.Socket.Close()
		}()

		for {
			conn, err := ctx.Socket.AcceptUnix()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					log.Printf("[listener] closed")
					return nil
				}
				continue
			}
			group.Go(func() error {
				RunCommandHandler(groupCtx, ctx, conn)
				return nil
			})
		}
	})

	for _, target := range ctx.Targets {
		target.Start(group, groupCtx, ctx)
	}

	if ctx.PluginHosts != nil {
		ctx.PluginHosts.Start(group, groupCtx, ctx.Config)
	}

	ctx.Conntrack.Start(group, groupCtx)

	for _, frontend := range ctx.Frontends {
		frontend.Start(group, groupCtx)
	}
}

type PluginTunnel struct {
	Name        string
	Instance    int
	Tunnel      string
	ConnectedAt time.Time
	Remote      string
}

func (ctx *RuntimeContext) RegisterPluginTunnel(t PluginTunnel) {
	ctx.pluginMu.Lock()
	defer ctx.pluginMu.Unlock()
	ctx.pluginTunnels[t.Tunnel] = t
}
