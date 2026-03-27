package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"proxy-gateway/api"
	"proxy-gateway/ipc"
	"time"
)

func RunCommandHandler(groupCtx context.Context, ctx *RuntimeContext, conn *net.UnixConn) {
	defer conn.Close()

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	var request ipc.Wrapper
	err := decoder.Decode(&request)
	if err != nil {
		log.Printf("[ipc] decode request: %v", err)
		return
	}

	var response ipc.Wrapper
	switch request.Kind {
	case ipc.KindStatusRequest:
		result := ipc.StatusResponse{
			Targets:   make(map[string]ipc.StatusDetails),
			Frontends: make(map[string]ipc.StatusDetails),
		}
		for name, target := range ctx.Targets {
			state, err := target.State()
			statusErr := ""
			if err != nil {
				statusErr = err.Error()
			}
			result.Targets[name] = ipc.StatusDetails{State: state.String(), Err: statusErr}
		}
		for name, frontend := range ctx.Frontends {
			state, err := frontend.State()
			statusErr := ""
			if err != nil {
				statusErr = err.Error()
			}
			result.Frontends[name] = ipc.StatusDetails{State: state.String(), Err: statusErr}
		}
		response = ipc.WrapValue(result)
	case ipc.KindExternalFrontendRegisterRequest:
		response = ipc.WrapValue(ipc.ExternalFrontendRegisterResponse{
			Accepted: false,
			Message:  "external frontend registration not enabled in v1 runtime",
		})
	case ipc.KindExternalFrontendHeartbeatRequest:
		response = ipc.WrapValue(ipc.ExternalFrontendHeartbeatResponse{
			Accepted: false,
			Message:  "external frontend heartbeat not enabled in v1 runtime",
		})
	case ipc.KindPluginHostHelloRequest:
		hello, err := ipc.UnwrapValue[ipc.PluginHostHelloRequest](request)
		if err != nil {
			response = ipc.WrapValue(ipc.Error{Message: fmt.Sprintf("decode plugin hello: %v", err)})
			break
		}
		ctx.RegisterPluginTunnel(PluginTunnel{
			Name:        hello.PluginName,
			Instance:    hello.Instance,
			Tunnel:      hello.Tunnel,
			ConnectedAt: time.Now(),
			Remote:      conn.RemoteAddr().String(),
		})
		response = ipc.WrapValue(ipc.PluginHostHelloResponse{Accepted: true, Message: "registered"})
	default:
		response = ipc.WrapValue(ipc.Error{
			Message: fmt.Sprintf("unknown IPC kind: %v", request.Kind),
		})
	}
	_ = encoder.Encode(response)
}

func RunCommand(cfg *api.Config, command string) error {
	socketAddr, err := net.ResolveUnixAddr("unix", cfg.Runtime.SocketPath)
	if err != nil {
		return fmt.Errorf("resolve unix socket address: %w", err)
	}
	conn, err := net.DialUnix("unix", nil, socketAddr)
	if err != nil {
		return fmt.Errorf("dial unix socket: %w", err)
	}
	defer conn.Close()

	var request ipc.Wrapper
	switch command {
	case "status":
		request = ipc.WrapValue(ipc.StatusRequest{})
	default:
		return fmt.Errorf("unknown command %q", command)
	}

	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	if err := encoder.Encode(request); err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	var response ipc.Wrapper
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	switch response.Kind {
	case ipc.KindError:
		body, err := ipc.UnwrapValue[ipc.Error](response)
		if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}
		fmt.Println(body)
	case ipc.KindStatusResponse:
		body, err := ipc.UnwrapValue[ipc.StatusResponse](response)
		if err != nil {
			return fmt.Errorf("decode status: %w", err)
		}
		fmt.Print(body.ConsoleString())
	default:
		return fmt.Errorf("unexpected response kind %d", response.Kind)
	}

	return nil
}
