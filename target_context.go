package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"proxy-gateway/api"
	"proxy-gateway/misc"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

type TargetServiceContext struct {
	name     string
	protocol api.Protocol
	dst      netip.AddrPort
	frontend *FrontendContext // set by frontend
}

type TargetContext struct {
	name            string
	services        map[string]*TargetServiceContext
	idleTimeout     api.TTL
	activator       api.Activator
	activatorConfig map[string]any
	state           atomic.Int32
	err             misc.AtomicBox[error]
	requests        chan api.TargetState
	groupCtx        misc.AtomicBox[context.Context]
}

func (t *TargetContext) Name() string {
	return t.name
}

func (t *TargetContext) State() (api.TargetState, error) {
	return api.TargetState(t.state.Load()), t.Error()
}

func (t *TargetContext) Error() error {
	return t.err.Load()
}

func (t *TargetContext) Activate() bool {
	groupCtx := t.groupCtx.Load()
	if groupCtx == nil {
		return false
	}

	state := api.TargetState(t.state.Load())
	switch state {
	case api.TargetStateActive, api.TargetStateWarming:
		return true
	case api.TargetStateDraining:
		return false
	case api.TargetStateInactive:
		ok := t.state.CompareAndSwap(int32(api.TargetStateInactive), int32(api.TargetStateWarming))
		if ok {
			t.err.Store(nil)
			select {
			case <-groupCtx.Done():
				return false
			case t.requests <- api.TargetStateWarming:
				return ok
			}
		}
		return ok
	default:
		panic(fmt.Sprintf("unknown target state: %d", state))
	}
}

func (t *TargetContext) Deactivate() bool {
	groupCtx := t.groupCtx.Load()
	if groupCtx == nil {
		return false
	}

	state := api.TargetState(t.state.Load())
	switch state {
	case api.TargetStateInactive, api.TargetStateDraining:
		return true
	case api.TargetStateWarming:
		return false
	case api.TargetStateActive:
		ok := t.state.CompareAndSwap(int32(api.TargetStateActive), int32(api.TargetStateDraining))
		if ok {
			t.err.Store(nil)
			select {
			case <-groupCtx.Done():
				return false
			case t.requests <- api.TargetStateDraining:
				return ok
			}
		}
		return ok
	default:
		panic(fmt.Sprintf("unknown target state: %d", state))
	}
}

func LoadTargetContext(rt *RuntimeContext, cfg api.TargetConfig) (*TargetContext, error) {
	var ok bool
	var activator api.Activator
	var activatorConfig map[string]any

	if cfg.Activator != nil {
		activator, ok = rt.Activators[cfg.Activator.Kind]
		if !ok {
			return nil, fmt.Errorf("unknown activator kind %q", cfg.Activator.Kind)
		}
		activatorConfig = cfg.Activator.Config
	}

	t := &TargetContext{
		name:            cfg.Name,
		services:        make(map[string]*TargetServiceContext, len(cfg.Services)),
		idleTimeout:     cfg.IdleTimeout,
		activator:       activator,
		activatorConfig: activatorConfig,
		requests:        make(chan api.TargetState, 1),
	}
	for _, service := range cfg.Services {
		if !ok {
			return nil, fmt.Errorf("no frontend for service %q", service.Name)
		}
		t.services[service.Name] = &TargetServiceContext{
			name:     service.Name,
			protocol: service.Protocol,
			dst:      service.Address,
		}
	}
	t.state.Store(int32(api.TargetStateInactive))
	return t, nil
}

func (t *TargetContext) Start(group *errgroup.Group, groupCtx context.Context, rt *RuntimeContext) {
	groupCtx, cancel := context.WithCancel(groupCtx)
	t.groupCtx.Store(groupCtx)

	group.Go(func() error {
		defer cancel()
		defer func() {
			log.Printf("[target:%s] management stopped", t.name)
		}()
		log.Printf("[target:%s] management started", t.name)
		for {
			select {
			case <-groupCtx.Done():
				return nil
			case next := <-t.requests:
				switch next {
				case api.TargetStateWarming:
					activationStarted := time.Now()
					log.Printf("[target:%s] activator started", t.name)
					err := t.activator.OnActivate(t.activatorConfig)
					log.Printf("[target:%s] activator finished duration=%s", t.name, time.Since(activationStarted))
					t.err.Store(err)
					if err == nil {
						// enable DNAT for each frontend-backend route
						for serviceName, s := range t.services {
							err = rt.Nftables.AddDNAT(s.protocol, s.frontend.listen, s.dst)
							if err != nil {
								t.err.Store(fmt.Errorf("failed to add DNAT for service %q: %v", serviceName, err))
							}
						}
						t.state.Store(int32(api.TargetStateActive))
					} else {
						t.state.Store(int32(api.TargetStateInactive))
					}
				case api.TargetStateDraining:
					deactivationStarted := time.Now()
					log.Printf("[target:%s] deactivator started", t.name)
					err := t.activator.OnDeactivate(t.activatorConfig)
					log.Printf("[target:%s] deactivator finished duration=%s", t.name, time.Since(deactivationStarted))
					t.err.Store(err)
					// disable DNAT for each frontend
					for serviceName, s := range t.services {
						err = rt.Nftables.ClearDNAT(s.protocol, s.frontend.listen)
						if err != nil {
							t.err.Store(fmt.Errorf("failed to clear DNAT for service %q: %v", serviceName, err))
						}
					}
					t.state.Store(int32(api.TargetStateInactive))
				default:
					t.state.Store(int32(next))
				}
			}
		}
	})
}
