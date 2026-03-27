package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"proxy-gateway/api"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"
)

type PluginProcess struct {
	Name       string
	Instance   int
	Command    string
	Args       []string
	Env        map[string]string
	WorkDir    string
	TunnelName string
}

type PluginHostContext struct {
	processes []PluginProcess
}

func LoadPluginHostContext(cfg *api.Config) (*PluginHostContext, error) {
	ctx := &PluginHostContext{}

	hostsByName := map[string]api.PluginHostProcessConfig{}
	loadOrder := []string{cfg.Runtime.PluginDefinitionsPath}
	for _, dir := range loadOrder {
		hosts, err := loadPluginHostsFromDir(dir)
		if err != nil {
			return nil, err
		}
		for _, host := range hosts {
			hostsByName[host.Name] = host
		}
	}

	hostNames := make([]string, 0, len(hostsByName))
	for name := range hostsByName {
		hostNames = append(hostNames, name)
	}
	sort.Strings(hostNames)

	for _, name := range hostNames {
		host := hostsByName[name]
		if !host.IsAutoStart() {
			continue
		}
		for i := 0; i < host.Instances; i++ {
			tunnelName := fmt.Sprintf("%s-%d", host.Name, i)
			ctx.processes = append(ctx.processes, PluginProcess{
				Name:       host.Name,
				Instance:   i,
				Command:    host.Command,
				Args:       append([]string{}, host.Args...),
				Env:        host.Env,
				WorkDir:    host.WorkDir,
				TunnelName: tunnelName,
			})
		}
	}

	return ctx, nil
}

func loadPluginHostsFromDir(dir string) ([]api.PluginHostProcessConfig, error) {
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugin-host config dir %q: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var hosts []api.PluginHostProcessConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read plugin-host config %q: %w", path, err)
		}
		fileCfg, err := api.ParsePluginHostConfig(data)
		if err != nil {
			return nil, fmt.Errorf("parse plugin-host config %q: %w", path, err)
		}
		hosts = append(hosts, fileCfg.Hosts...)
	}
	return hosts, nil
}

func (ctx *PluginHostContext) Start(group *errgroup.Group, groupCtx context.Context, cfg *api.Config) {
	for _, process := range ctx.processes {
		proc := process
		group.Go(func() error {
			return runPluginProcess(groupCtx, cfg, proc)
		})
	}
}

func runPluginProcess(groupCtx context.Context, cfg *api.Config, process PluginProcess) error {
	args := append([]string{}, process.Args...)
	cmd := exec.CommandContext(groupCtx, process.Command, args...)
	if process.WorkDir != "" {
		cmd.Dir = process.WorkDir
	}

	env := os.Environ()
	env = append(env,
		"PROXY_GATEWAY_RUNTIME_SOCKET="+cfg.Runtime.SocketPath,
		"PROXY_GATEWAY_PLUGIN_TUNNEL="+process.TunnelName,
		"PROXY_GATEWAY_PLUGIN_NAME="+process.Name,
		fmt.Sprintf("PROXY_GATEWAY_PLUGIN_INSTANCE=%d", process.Instance),
	)
	for key, value := range process.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("[plugin-host:%s#%d] starting command=%s", process.Name, process.Instance, process.Command)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start plugin-host %s#%d: %w", process.Name, process.Instance, err)
	}
	log.Printf("[plugin-host:%s#%d] pid=%d", process.Name, process.Instance, cmd.Process.Pid)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("plugin-host %s#%d exited with error: %w", process.Name, process.Instance, err)
	}
	log.Printf("[plugin-host:%s#%d] exited", process.Name, process.Instance)
	return nil
}
