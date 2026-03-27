package contracts

import (
	"context"
	"net/netip"
	"proxy-gateway/api"
)

// TargetLifecycle is the minimal activation surface that other subsystems need.
type TargetLifecycle interface {
	Name() string
	Activate() bool
	Deactivate() bool
	State() (api.TargetState, error)
}

// TargetLookup resolves a target by name and keeps callers decoupled from storage details.
type TargetLookup interface {
	GetTarget(name string) (TargetLifecycle, bool)
}

// FrontendEvents captures only the events emitted by listeners/interceptors.
type FrontendEvents interface {
	HintActivate(target string)
	FrontendState(name string) (api.FrontendState, error)
}

// NATManager is the minimal API that targets require from nftables.
type NATManager interface {
	AddDNAT(protocol api.Protocol, matchAddress netip.AddrPort, targetAddress netip.AddrPort) error
	ClearDNAT(protocol api.Protocol, matchAddress netip.AddrPort) error
	SetTTL(name string, protocol api.Protocol, matchListen netip.AddrPort, ttl api.TTL) error
}

// ConntrackReadModel is the minimal data shape required by conntrack scans.
type ConntrackReadModel interface {
	ListTargets() []TargetSnapshot
}

// TargetSnapshot is a read-only view used by watchdog logic.
type TargetSnapshot interface {
	Lifecycle() TargetLifecycle
	IdleTimeout() api.TTL
	Services() []ServiceSnapshot
}

// ServiceSnapshot is a read-only view used by flow matching logic.
type ServiceSnapshot interface {
	Protocol() api.Protocol
	Destination() netip.AddrPort
}

// PluginHost is an external worker that can emit activation hints.
type PluginHost interface {
	Name() string
	Start(context.Context, ActivationSink) error
}

// ActivationSink is the single method plugin hosts need to call.
type ActivationSink interface {
	HintActivate(targetName string)
}
