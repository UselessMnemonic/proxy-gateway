package main

import (
	"fmt"
	"plugin"
	"proxy-gateway/api"
)

type PluginContext struct {
	Path       string
	Handle     *plugin.Plugin
	OnRegister api.OnRegisterFunc
}

func LoadPluginContext(path string) (*PluginContext, error) {
	handle, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open plugin %q: %w", path, err)
	}

	symbol, err := handle.Lookup(api.SymbolOnRegister)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s#%s: %w", path, api.SymbolOnRegister, err)
	}
	onRegister, ok := symbol.(api.OnRegisterFunc)
	if !ok {
		return nil, fmt.Errorf("could not find %s in %s", api.SymbolOnRegister, path)
	}

	pluginCtx := &PluginContext{
		Path:       path,
		Handle:     handle,
		OnRegister: onRegister,
	}
	return pluginCtx, nil
}
