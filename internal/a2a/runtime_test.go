package a2a

import (
	"context"
	"testing"

	"github.com/nikstern/swarmies"
	"google.golang.org/genai"
)

type claimRecorder struct {
	claimed []string
}

func (c *claimRecorder) Claim(_ context.Context, id string) error {
	c.claimed = append(c.claimed, id)
	return nil
}

func TestNewRuntimeConfiguresADKLoaderAndSessionService(t *testing.T) {
	t.Parallel()

	runtime, err := NewRuntime("generalist", &claimRecorder{})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	cfg := runtime.LauncherConfig()
	if cfg == nil {
		t.Fatal("LauncherConfig() = nil")
	}
	if cfg.AgentLoader == nil {
		t.Fatal("LauncherConfig().AgentLoader = nil")
	}
	if got := cfg.AgentLoader.RootAgent().Name(); got != "generalist" {
		t.Fatalf("root agent name = %q, want %q", got, "generalist")
	}
	if cfg.SessionService == nil {
		t.Fatal("LauncherConfig().SessionService = nil")
	}
	if runtime.Launcher() == nil {
		t.Fatal("Launcher() = nil")
	}
}

func TestGatewayDispatchClaimsTaskAndReturnsStructuredResult(t *testing.T) {
	t.Parallel()

	claims := &claimRecorder{}
	runtime, err := NewRuntime("generalist", claims)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	gateway := NewGateway(runtime)
	result, err := gateway.Dispatch(context.Background(), swarmies.DispatchRequest{
		TaskID:    "swarmies-0kq",
		ContextID: "swarmies-0kq",
		Profile:   swarmies.AgentProfile{ID: swarmies.ProfileGeneralist},
		WorkItem: swarmies.WorkItem{
			TaskID: "swarmies-0kq",
			Title:  "Expose a minimal ADK-backed generalist agent over A2A",
		},
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if len(claims.claimed) != 1 || claims.claimed[0] != "swarmies-0kq" {
		t.Fatalf("claimed = %v, want [swarmies-0kq]", claims.claimed)
	}
	if result.TaskID != "swarmies-0kq" {
		t.Fatalf("result.TaskID = %q, want %q", result.TaskID, "swarmies-0kq")
	}
	if result.State != swarmies.StateSucceeded {
		t.Fatalf("result.State = %q, want %q", result.State, swarmies.StateSucceeded)
	}
	if result.Summary == "" {
		t.Fatal("result.Summary = empty")
	}
}

func TestParseAgentWorkRequestRequiresTaskID(t *testing.T) {
	t.Parallel()

	_, err := parseAgentWorkRequest(&genai.Content{
		Parts: []*genai.Part{
			{Text: `{"context_id":"ctx-1"}`},
		},
	})
	if err == nil {
		t.Fatal("parseAgentWorkRequest() error = nil, want error")
	}
}
