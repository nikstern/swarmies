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
	gateway := a2a.NewGateway()
	policy := dispatch.NewDefaultResultPolicy()
	runtime := a2a.NewRuntime("generalist")
	dispatcher := dispatch.NewDispatcher(beadsClient, agentRegistry, gateway, policy)

	if err := dispatcher.RunOnce(ctx); err != nil {
		log.Fatalf("swarmies runtime (%s): %v", runtime.Description(), err)
	}
}
