package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"proxy-gateway/api"
	"proxy-gateway/ipc"

	"github.com/alecthomas/kingpin/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

func main() {
	ipc.RegisterGobTypes()

	app := kingpin.New("proxy-gateway", "Proxy Gateway runtime and control CLI")
	configPath := app.Flag("config", "Path to configuration file").Required().String()

	startCmd := app.Command("start", "Start the proxy gateway runtime")

	ctlCmd := app.Command("ctl", "Send IPC control requests to a running runtime")
	ctlStatusCmd := ctlCmd.Command("status", "Print runtime status")

	selected := kingpin.MustParse(app.Parse(os.Args[1:]))
	cfg, err := readConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	switch selected {
	case startCmd.FullCommand():
		if err := runStart(cfg); err != nil {
			log.Fatal(err)
		}
	case ctlStatusCmd.FullCommand():
		if err := RunCommand(cfg, "status"); err != nil {
			log.Fatal(err)
		}
	}
}

func runStart(cfg *api.Config) error {
	runtimeCtx, err := LoadRuntimeContext(cfg)
	if err != nil {
		return fmt.Errorf("load runtime context: %w", err)
	}

	rootCtx, deregister := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer deregister()

	group, groupCtx := errgroup.WithContext(rootCtx)
	group.SetLimit(-1)
	runtimeCtx.Start(group, groupCtx)

	<-rootCtx.Done()
	log.Printf("Shutting down...")
	return group.Wait()
}

func readConfig(path string) (*api.Config, error) {
	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg, err := api.ParseConfig(configData)
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}
