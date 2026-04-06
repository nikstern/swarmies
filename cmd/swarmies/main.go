package main

import (
	"context"
	"log"

	"github.com/nikstern/swarmies/internal/a2a"
	"github.com/nikstern/swarmies/internal/beads"
	"github.com/nikstern/swarmies/internal/dispatch"
	"github.com/nikstern/swarmies/internal/registry"
)

func main() {
	ctx := context.Background()

	beadsClient := beads.NewClient(beads.DefaultBinary)
	agentRegistry := registry.NewStatic(registry.DefaultProfiles()...)
	policy := dispatch.NewDefaultResultPolicy()
	runtime, err := a2a.NewRuntime("generalist", beadsClient)
	if err != nil {
		log.Fatalf("configure ADK runtime: %v", err)
	}
	gateway := a2a.NewGateway(runtime)
	dispatcher := dispatch.NewDispatcher(beadsClient, agentRegistry, gateway, policy)

	if err := dispatcher.RunOnce(ctx); err != nil {
		log.Fatalf("swarmies runtime (%s): %v", runtime.Description(), err)
	}
}
