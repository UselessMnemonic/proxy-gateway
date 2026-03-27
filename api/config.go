package api

import (
	"errors"
	"fmt"
	"net/netip"

	"gopkg.in/yaml.v3"
)

// ConfigVersionV1 is the supported schema version for configuration files.
const ConfigVersionV1 = "v1"

// Config is the root runtime configuration document.
type Config struct {
	Version   string           `yaml:"version"`
	Runtime   RuntimeConfig    `yaml:"runtime"`
	Targets   []TargetConfig   `yaml:"targets"`
	Frontends []FrontendConfig `yaml:"frontends"`
}

// RuntimeConfig configures process-level runtime behavior.
type RuntimeConfig struct {
	PluginDirectories []string `yaml:"plugin_directories"`
	SocketPath        string   `yaml:"socket_path"`
}

// TargetConfig defines a backend target and how it is activated.
type TargetConfig struct {
	Name        string                `yaml:"name"`
	Services    []TargetServiceConfig `yaml:"target_services"`
	IdleTimeout TTL                   `yaml:"idle_timeout"`
	Activator   *ActivatorConfig      `yaml:"activator"`
}

// TargetServiceConfig defines an addressable backend service on a target.
type TargetServiceConfig struct {
	Name     string         `yaml:"name"`
	Protocol Protocol       `yaml:"protocol"`
	Address  netip.AddrPort `yaml:"address"`
}

// FrontendConfig defines a listening socket and forwarding behavior.
type FrontendConfig struct {
	Name        string           `yaml:"name"`
	Protocol    Protocol         `yaml:"protocol"`
	Listen      netip.AddrPort   `yaml:"listen"`
	FlowTimeout TTL              `yaml:"flow_timeout"`
	Forward     ForwardConfig    `yaml:"forward"`
	Intercept   *InterceptConfig `yaml:"intercept"`
}

// ForwardConfig references which target service a frontend should route to.
type ForwardConfig struct {
	Target  string `yaml:"target"`
	Service string `yaml:"service"`
}

// ActivatorConfig selects an activator implementation and its configuration.
type ActivatorConfig struct {
	Kind   string         `yaml:"kind"`
	Config map[string]any `yaml:"config"`
}

// InterceptConfig selects an interceptor implementation and its configuration.
type InterceptConfig struct {
	Kind   string         `yaml:"kind"`
	Config map[string]any `yaml:"config"`
}

// ParseConfig decodes and validates YAML configuration bytes.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.normalizeOptionalMaps()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) normalizeOptionalMaps() {
	for i := range c.Targets {
		target := &c.Targets[i]
		if target.Activator != nil && target.Activator.Config == nil {
			target.Activator.Config = map[string]any{}
		}
	}

	for i := range c.Frontends {
		frontend := &c.Frontends[i]
		if frontend.Intercept != nil && frontend.Intercept.Config == nil {
			frontend.Intercept.Config = map[string]any{}
		}
	}
}

// Validate checks that the configuration is internally consistent and usable.
func (c *Config) Validate() error {
	if c.Version == "" {
		return errors.New("config version is required")
	}
	if c.Version != ConfigVersionV1 {
		return fmt.Errorf("unsupported config version %q", c.Version)
	}

	targets := make(map[string]map[string]TargetServiceConfig, len(c.Targets))
	for i, target := range c.Targets {
		targetRef := configItemRef("targets", i, target.Name)
		if target.Name == "" {
			return fmt.Errorf("%s.name is required", targetRef)
		}
		if _, exists := targets[target.Name]; exists {
			return fmt.Errorf("%s.name %q is duplicated", targetRef, target.Name)
		}
		if len(target.Services) == 0 {
			return fmt.Errorf("%s.target_services must contain at least one service", targetRef)
		}
		if target.Activator == nil {
			return fmt.Errorf("%s.activator is required", targetRef)
		}

		serviceMap := make(map[string]TargetServiceConfig, len(target.Services))
		for j, service := range target.Services {
			serviceRef := configItemRef(fmt.Sprintf("%s.target_services", targetRef), j, service.Name)
			if service.Name == "" {
				return fmt.Errorf("%s.name is required", serviceRef)
			}
			if _, exists := serviceMap[service.Name]; exists {
				return fmt.Errorf("%s.name %q is duplicated", serviceRef, service.Name)
			}
			if !service.Protocol.IsValid() {
				return fmt.Errorf("%s.protocol is invalid", serviceRef)
			}
			if !service.Address.IsValid() {
				return fmt.Errorf("%s.address is invalid", serviceRef)
			}
			if !service.Address.Addr().Is4() && !service.Address.Addr().Is6() {
				return fmt.Errorf("%s.address type must be IPv4 or IPv6", serviceRef)
			}
			if service.Address.Addr().Zone() != "" {
				return fmt.Errorf("%s.address cannot have zone", serviceRef)
			}
			serviceMap[service.Name] = service
		}
		targets[target.Name] = serviceMap

		if target.Activator.Kind == "" {
			return fmt.Errorf("%s.activator.kind is required", targetRef)
		}
	}

	frontends := make(map[string]struct{}, len(c.Frontends))
	for i, frontend := range c.Frontends {
		frontendRef := configItemRef("frontends", i, frontend.Name)
		if frontend.Name == "" {
			return fmt.Errorf("%s.name is required", frontendRef)
		}
		if _, exists := frontends[frontend.Name]; exists {
			return fmt.Errorf("%s.name %q is duplicated", frontendRef, frontend.Name)
		}
		frontends[frontend.Name] = struct{}{}

		if !frontend.Listen.IsValid() {
			return fmt.Errorf("%s.listen is invalid", frontendRef)
		}
		if !frontend.Listen.Addr().Is4() && !frontend.Listen.Addr().Is6() {
			return fmt.Errorf("%s.listen type must be IPv4 or IPv6", frontendRef)
		}
		if frontend.Forward.Target == "" {
			return fmt.Errorf("%s.forward.target is required", frontendRef)
		}
		if frontend.Forward.Service == "" {
			return fmt.Errorf("%s.forward.service is required", frontendRef)
		}
		targetServices, exists := targets[frontend.Forward.Target]
		if !exists {
			return fmt.Errorf("%s.forward.target %q does not match a configured target", frontendRef, frontend.Forward.Target)
		}
		service, exists := targetServices[frontend.Forward.Service]
		if !exists {
			return fmt.Errorf(
				"%s.forward.service %q does not match a service on target %q",
				frontendRef,
				frontend.Forward.Service,
				frontend.Forward.Target,
			)
		}
		if frontend.Protocol != service.Protocol {
			return fmt.Errorf(
				"%s.protocol %q does not match target service protocol %q",
				frontendRef,
				frontend.Protocol.String(),
				service.Protocol.String(),
			)
		}
		if frontend.Intercept == nil {
			return fmt.Errorf("%s.intercept is required", frontendRef)
		}
		if frontend.Intercept.Kind == "" {
			return fmt.Errorf("%s.intercept.kind is required", frontendRef)
		}
	}

	return nil
}

func configItemRef(section string, index int, name string) string {
	if name != "" {
		return fmt.Sprintf("%s[%q]", section, name)
	}
	return fmt.Sprintf("%s[%d]", section, index)
}
