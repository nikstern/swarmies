package a2a

import (
	"context"
	"fmt"

	"github.com/nikstern/swarmies"
)

type Gateway struct{}

func NewGateway() *Gateway {
	return &Gateway{}
}

func (g *Gateway) Dispatch(context.Context, swarmies.DispatchRequest) (swarmies.DispatchResult, error) {
	return swarmies.DispatchResult{
		TaskID:  "",
		State:   swarmies.StateSubmitted,
		Summary: "A2A dispatch scaffold is not implemented yet",
	}, nil
}

type Runtime struct {
	Name string
}

func NewRuntime(name string) *Runtime {
	if name == "" {
		name = "generalist"
	}

	return &Runtime{Name: name}
}

func (r *Runtime) Description() string {
	return fmt.Sprintf("ADK-backed runtime scaffold for profile %q", r.Name)
}
