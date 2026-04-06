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
	gateway := a2a.NewGateway()
	dispatcher := dispatch.NewDispatcher(beadsClient, agentRegistry, gateway, policy)

	if err := dispatcher.RunOnce(ctx); err != nil {
		log.Fatalf("swarmies runtime: %v", err)
	}
}
