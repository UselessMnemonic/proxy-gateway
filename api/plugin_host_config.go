package api

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// PluginHostFile stores one or more plugin host process definitions.
type PluginHostFile struct {
	Version string                    `yaml:"version"`
	Hosts   []PluginHostProcessConfig `yaml:"plugin_hosts"`
}

// PluginHostProcessConfig configures an external plugin host subprocess.
type PluginHostProcessConfig struct {
	Name      string            `yaml:"name"`
	Command   string            `yaml:"command"`
	Args      []string          `yaml:"args"`
	Env       map[string]string `yaml:"env"`
	WorkDir   string            `yaml:"work_dir"`
	Instances int               `yaml:"instances"`
	AutoStart *bool             `yaml:"auto_start"`
}

func ParsePluginHostConfig(data []byte) (*PluginHostFile, error) {
	var cfg PluginHostFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse plugin-host config: %w", err)
	}
	if cfg.Version == "" {
		cfg.Version = ConfigVersionV1
	}
	if cfg.Version != ConfigVersionV1 {
		return nil, fmt.Errorf("unsupported plugin-host config version %q", cfg.Version)
	}
	for i := range cfg.Hosts {
		host := &cfg.Hosts[i]
		if host.Name == "" {
			return nil, fmt.Errorf("plugin_hosts[%d].name is required", i)
		}
		if host.Command == "" {
			return nil, fmt.Errorf("plugin_hosts[%q].command is required", host.Name)
		}
		if host.Instances <= 0 {
			host.Instances = 1
		}
		if host.Env == nil {
			host.Env = map[string]string{}
		}
	}
	return &cfg, nil
}

func (p PluginHostProcessConfig) IsAutoStart() bool {
	if p.AutoStart == nil {
		return true
	}
	return *p.AutoStart
}
