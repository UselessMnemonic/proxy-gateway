package model

import (
	"fmt"
	"net/netip"
	"proxy-gateway/api"
)

// ResourceKind models a Teleport/nft-style control resource envelope.
type ResourceKind string

const (
	ResourceKindTarget   ResourceKind = "target"
	ResourceKindFrontend ResourceKind = "frontend"
	ResourceKindHost     ResourceKind = "plugin_host"
)

// Metadata is shared by all externally-managed resources.
type Metadata struct {
	Name    string            `json:"name" yaml:"name"`
	Labels  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Version string            `json:"version,omitempty" yaml:"version,omitempty"`
}

// Resource is a discriminated union used for apply/replace operations.
type Resource struct {
	Kind     ResourceKind `json:"kind" yaml:"kind"`
	Metadata Metadata     `json:"metadata" yaml:"metadata"`
	Spec     any          `json:"spec" yaml:"spec"`
}

// TargetSpec describes a backend target and its routable ports.
type TargetSpec struct {
	Activator   api.ActivatorConfig `json:"activator" yaml:"activator"`
	IdleTimeout api.TTL             `json:"idle_timeout" yaml:"idle_timeout"`
	Ports       []PortSpec          `json:"ports" yaml:"ports"`
}

// PortSpec describes one target port/protocol tuple.
type PortSpec struct {
	Name     string         `json:"name" yaml:"name"`
	Protocol api.Protocol   `json:"protocol" yaml:"protocol"`
	Address  netip.AddrPort `json:"address" yaml:"address"`
}

// FrontendSpec describes listener behavior and a route to a target port.
type FrontendSpec struct {
	Protocol    api.Protocol         `json:"protocol" yaml:"protocol"`
	Listen      netip.AddrPort       `json:"listen" yaml:"listen"`
	FlowTimeout api.TTL              `json:"flow_timeout" yaml:"flow_timeout"`
	Intercept   api.InterceptConfig  `json:"intercept" yaml:"intercept"`
	Route       FrontendRoute        `json:"route" yaml:"route"`
	HostedBy    FrontendHostingModel `json:"hosted_by" yaml:"hosted_by"`
}

// FrontendRoute links a frontend to an externally managed target and port.
type FrontendRoute struct {
	Target string `json:"target" yaml:"target"`
	Port   string `json:"port" yaml:"port"`
}

// FrontendHostingModel allows in-process or external listener ownership.
type FrontendHostingModel string

const (
	FrontendHostedInternal FrontendHostingModel = "internal"
	FrontendHostedExternal FrontendHostingModel = "external"
)

// PluginHostSpec defines an external host process lifecycle contract.
type PluginHostSpec struct {
	Driver string         `json:"driver" yaml:"driver"`
	Config map[string]any `json:"config" yaml:"config"`
}

// ApplyRequest updates the running desired state.
type ApplyRequest struct {
	Resources []Resource `json:"resources" yaml:"resources"`
	Prune     bool       `json:"prune" yaml:"prune"`
}

func (r Resource) Validate() error {
	if r.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	switch r.Kind {
	case ResourceKindTarget, ResourceKindFrontend, ResourceKindHost:
		return nil
	default:
		return fmt.Errorf("unsupported resource kind %q", r.Kind)
	}
}
