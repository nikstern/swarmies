package a2a

import (
	"context"
	"strings"
	"testing"

	"github.com/nikstern/swarmies"
	"google.golang.org/genai"
)

type claimRecorder struct {
	claimed []string
	reports []string
	notes   []string
}

func (c *claimRecorder) Claim(_ context.Context, id string) error {
	c.claimed = append(c.claimed, id)
	return nil
}

func (c *claimRecorder) Comment(_ context.Context, _ string, body string) error {
	c.notes = append(c.notes, body)
	return nil
}

func (c *claimRecorder) Note(_ context.Context, _ string, body string) error {
	c.reports = append(c.reports, body)
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

func TestTriageWorkRoutesImplementationTasksToCoding(t *testing.T) {
	t.Parallel()

	result, note := triageWork(agentWorkRequest{
		TaskID:    "swarmies-1xm",
		ContextID: "swarmies-1xm",
		Profile:   "generalist",
		WorkItem: swarmies.WorkItem{
			Title: "Implement dispatcher outcome handling",
			Body:  "Write code and tests for non-success runtime handling",
		},
	})

	if result.Outcome != swarmies.OutcomeHandoff {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, swarmies.OutcomeHandoff)
	}
	if result.Handoff == nil || result.Handoff.TargetProfile != swarmies.ProfileCoding {
		t.Fatalf("Handoff = %#v, want coding", result.Handoff)
	}
	if !strings.Contains(note, "Recommended profile: coding") {
		t.Fatalf("note = %q, want coding recommendation", note)
	}
}

func TestTriageWorkMarksBlockedTasks(t *testing.T) {
	t.Parallel()

	result, note := triageWork(agentWorkRequest{
		TaskID:    "swarmies-1xm",
		ContextID: "swarmies-1xm",
		Profile:   "generalist",
		WorkItem: swarmies.WorkItem{
			Title: "Follow up after dependency lands",
			Body:  "This task is blocked by the upstream registry dependency.",
		},
	})

	if result.Outcome != swarmies.OutcomeBlocked {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, swarmies.OutcomeBlocked)
	}
	if result.BlockedReason == "" {
		t.Fatal("BlockedReason = empty")
	}
	if !strings.Contains(note, "Blocked reason:") {
		t.Fatalf("note = %q, want blocked reason", note)
	}
}

func TestTriageWorkLeavesActionablePlanningNote(t *testing.T) {
	t.Parallel()

	result, note := triageWork(agentWorkRequest{
		TaskID:    "swarmies-1xm",
		ContextID: "swarmies-1xm",
		Profile:   "generalist",
		WorkItem: swarmies.WorkItem{
			Title: "Document v1 triage workflow",
			Body:  "Capture the planner-oriented runtime contract for the team.",
		},
	})

	if result.Outcome != swarmies.OutcomeSuccess {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, swarmies.OutcomeSuccess)
	}
	if !strings.Contains(note, "Next step:") {
		t.Fatalf("note = %q, want next step", note)
	}
}

func TestTriageWorkMarksFailedExecution(t *testing.T) {
	t.Parallel()

	result, note := triageWork(agentWorkRequest{
		TaskID:    "swarmies-1xm",
		ContextID: "swarmies-1xm",
		Profile:   "generalist",
		WorkItem: swarmies.WorkItem{
			Title: "Fix the broken retry path",
			Body:  "The last execution failed with an error and needs inspection.",
		},
	})

	if result.Outcome != swarmies.OutcomeFailed {
		t.Fatalf("Outcome = %q, want %q", result.Outcome, swarmies.OutcomeFailed)
	}
	if result.ErrorMessage == "" {
		t.Fatal("ErrorMessage = empty")
	}
	if !strings.Contains(note, "Failure detail:") {
		t.Fatalf("note = %q, want failure detail", note)
	}
}

func TestExecutionNoteIncludesStructuredFields(t *testing.T) {
	t.Parallel()

	note := executionNote(swarmies.ExecutionResult{
		TaskID:    "swarmies-1xm",
		ContextID: "ctx-1",
		Outcome:   swarmies.OutcomeNeedsInput,
		Summary:   "Need product guidance before proceeding",
		InputRequest: &swarmies.InputRequest{
			Question: "Which API owns retries?",
			Details:  "Dispatcher and agent policy both mention retries.",
		},
		Details: map[string]any{
			"triage_outcome":        "needs_input",
			"recommended_next_step": "Add the missing ownership decision to the bead.",
		},
	})

	for _, want := range []string{
		"[swarmies/execution]",
		"status: needs_input",
		"input_question: Which API owns retries?",
		"recommended_next_step: Add the missing ownership decision to the bead.",
	} {
		if !strings.Contains(note, want) {
			t.Fatalf("executionNote() = %q, want substring %q", note, want)
		}
	}
}
