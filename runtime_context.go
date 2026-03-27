package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"proxy-gateway/api"
	"strings"

	"golang.org/x/sync/errgroup"
)

type RuntimeContext struct {
	Plugins      map[string]*PluginContext
	Targets      map[string]*TargetContext
	Frontends    map[string]*FrontendContext
	Activators   map[string]api.Activator
	Interceptors map[string]api.Interceptor
	Nftables     *NftablesContext
	Conntrack    *ConntrackContext
	Socket       *net.UnixListener
}

func LoadRuntimeContext(cfg *api.Config) (*RuntimeContext, error) {
	var err error

	ctx := &RuntimeContext{
		Plugins:      make(map[string]*PluginContext),
		Targets:      make(map[string]*TargetContext, len(cfg.Targets)),
		Frontends:    make(map[string]*FrontendContext, len(cfg.Frontends)),
		Activators:   DefaultActivators(),
		Interceptors: DefaultInterceptors(),
	}

	// in case something goes wrong here, clean up resources immediately
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

	// capture ipc socket
	var socketAddr *net.UnixAddr
	socketAddr, err = net.ResolveUnixAddr("unix", cfg.Runtime.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve unix socket address: %v", err)
	}
	ctx.Socket, err = net.ListenUnix("unix", socketAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind unix socket: %v", err)
	}
	ctx.Socket.SetUnlinkOnClose(true)

	// load plugins
	for _, directory := range cfg.Runtime.PluginDirectories {
		var entries []os.DirEntry
		var pluginCtx *PluginContext

		entries, err = os.ReadDir(directory)
		if err != nil {
			return nil, fmt.Errorf("error reading plugin directory %q: %w", directory, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if _, exists := ctx.Plugins[name]; exists {
				err = fmt.Errorf("plugin %q is defined more than once", name)
				return nil, err
			}
			path := filepath.Join(directory, entry.Name())

			pluginCtx, err = LoadPluginContext(path)
			if err != nil {
				return nil, fmt.Errorf("error loading plugin %q: %w", path, err)
			}
			ctx.Plugins[name] = pluginCtx

			err = pluginCtx.OnRegister(ctx)
			if err != nil {
				return nil, fmt.Errorf("error loading plugin %q: %w", path, err)
			}
		}
	}

	// load targets
	for _, targetCfg := range cfg.Targets {
		var targetCtx *TargetContext
		targetCtx, err = LoadTargetContext(ctx, targetCfg)
		if err != nil {
			return nil, fmt.Errorf("error loading target %q: %w", targetCfg.Name, err)
		}
		ctx.Targets[targetCfg.Name] = targetCtx
	}

	// load frontends
	for _, frontendCfg := range cfg.Frontends {
		var frontendCtx *FrontendContext
		frontendCtx, err = LoadFrontendContext(ctx, frontendCfg)
		if err != nil {
			return nil, fmt.Errorf("error loading frontend %q: %w", frontendCfg.Name, err)
		}
		ctx.Frontends[frontendCfg.Name] = frontendCtx
	}

	// load nftables
	ctx.Nftables, err = DialNftables()
	if err != nil {
		return nil, err
	}
	for fName, f := range ctx.Frontends {
		err = ctx.Nftables.SetTTL(fName, f.protocol, f.listen, f.flowTimeout)
		if err != nil {
			return nil, err
		}
	}

	// load conntrack watchdog
	ctx.Conntrack, err = DialConntrack(ctx)
	if err != nil {
		return nil, err
	}

	return ctx, err
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

	// command listener
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

	// target activators
	for _, target := range ctx.Targets {
		target.Start(group, groupCtx, ctx)
	}

	// conntrack watchdog
	ctx.Conntrack.Start(group, groupCtx)

	// frontend listeners
	for _, frontend := range ctx.Frontends {
		frontend.Start(group, groupCtx)
	}
}
