package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikstern/swarmies/internal/a2a"
	"github.com/nikstern/swarmies/internal/beads"
	"github.com/nikstern/swarmies/internal/dispatch"
	"github.com/nikstern/swarmies/internal/registry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	beadsClient := beads.NewClient(beads.DefaultBinary)

	if err := run(ctx, beadsClient, os.Args[1:]); err != nil {
		log.Fatalf("swarmies runtime: %v", err)
	}
}

func run(ctx context.Context, beadsClient *beads.Client, args []string) error {
	mode := "dispatch"
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		mode = args[0]
		args = args[1:]
	}

	switch mode {
	case "dispatch":
		return runDispatcher(ctx, beadsClient)
	case "agent":
		return runAgent(ctx, beadsClient, args)
	default:
		return fmt.Errorf("unknown mode %q; want dispatch or agent", mode)
	}
}

func runDispatcher(ctx context.Context, beadsClient *beads.Client) error {
	agentRegistry := registry.NewStatic(registry.DefaultProfiles()...)
	policy := dispatch.NewDefaultResultPolicy()
	gateway := a2a.NewGateway()
	dispatcher := dispatch.NewDispatcher(beadsClient, agentRegistry, gateway, policy)
	return dispatcher.RunOnce(ctx)
}

func runAgent(ctx context.Context, beadsClient *beads.Client, args []string) error {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	profile := fs.String("profile", "generalist", "agent profile name to expose")
	port := fs.Int("port", 8080, "port to listen on")
	publicURL := fs.String("public-url", "", "public base URL advertised in the agent card")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected extra arguments: %v", fs.Args())
	}

	runtime, err := a2a.NewRuntime(*profile, beadsClient)
	if err != nil {
		return err
	}

	return runtime.RunServer(ctx, a2a.ServerConfig{
		Port:      *port,
		PublicURL: *publicURL,
	})
}
