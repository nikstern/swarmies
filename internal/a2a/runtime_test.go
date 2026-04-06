package a2a

import (
	"context"
	"testing"

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
