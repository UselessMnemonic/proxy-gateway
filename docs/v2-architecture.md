# V2 Architecture Direction

This document captures the V2 direction for stronger subsystem encapsulation and package-friendly operations.

## Configuration model

The application takes exactly one bootstrap input: `--config <path>`.

From that file, runtime paths are defined explicitly. The runtime does not hard-code plugin lookup directories.

Required runtime fields in the main config:

- `runtime.socket_path`
- `runtime.plugin_definitions_path`

## Suggested APT layout (convention, not hard-coded)

A package can *choose* to install files using this layout:

1. Binary: `/usr/bin/proxy-gateway`
2. Main config: `/etc/proxy-gateway/proxy-gateway.yaml`
3. Plugin definitions: `/etc/proxy-gateway/plugins/*.yaml`

For user-installed plugins, user/admin drop-ins should be placed in the same configured definitions directory (for the suggested layout: `/etc/proxy-gateway/plugins/`).

## Updated decisions

- **Single plugin definition directory**: no `.d` suffix, no implied precedence semantics.
- **Plugin binaries are resolved from plugin definitions**; the runtime does not use a plugin library list key.
- **Plugin IPC is internal-only**. It is not exposed on the CLI surface.
- **Plugin hosts are subprocesses managed by the runtime** so package/service managers can inspect process trees.

## New runtime behavior

1. Runtime starts the main IPC socket.
2. Runtime loads plugin-host definitions from `plugin_definitions_path`.
3. Runtime starts configured plugin-host subprocesses during bringup.
4. Each subprocess dials runtime IPC and sends a plugin hello message (`PluginHostHelloRequest`) with plugin/tunnel identity.
5. Runtime tracks tunnel ownership in-memory and can distinguish plugin-host traffic from CLI traffic by message kind.
6. On shutdown, context cancellation tears down subprocesses.

## Next implementation steps

1. Move root subsystem implementations (`frontend`, `target`, `nftables`, `conntrack`) into dedicated packages and wire through `internal/v2/contracts`.
2. Add a runtime IPC lease table for external frontends and plugin tunnels.
3. Add authn/authz for non-local IPC peers where needed.
4. Add richer plugin-host drivers (process, socket, sidecar).
5. Build migration helpers from V1 config to V2 resource model.
