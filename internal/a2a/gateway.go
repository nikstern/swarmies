package a2a

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nikstern/swarmies"
	"google.golang.org/genai"
)

type Gateway struct {
	runtime *Runtime
}

func NewGateway(runtime *Runtime) *Gateway {
	return &Gateway{runtime: runtime}
}

func (g *Gateway) Dispatch(ctx context.Context, req swarmies.DispatchRequest) (swarmies.DispatchResult, error) {
	if g == nil || g.runtime == nil {
		return swarmies.DispatchResult{}, fmt.Errorf("a2a: runtime is not configured")
	}

	prompt, err := json.Marshal(agentWorkRequest{
		TaskID:    req.TaskID,
		ContextID: req.ContextID,
		Profile:   string(req.Profile.ID),
		WorkItem:  req.WorkItem,
	})
	if err != nil {
		return swarmies.DispatchResult{}, fmt.Errorf("a2a: encode request: %w", err)
	}

	return g.runtime.Run(ctx, req.ContextID, &genai.Content{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			{Text: string(prompt)},
		},
	})
}
