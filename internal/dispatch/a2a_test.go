package dispatch

import (
	"encoding/json"
	"testing"

	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/nikstern/swarmies"
)

func TestBuildMessageParamsEncodesWorkItem(t *testing.T) {
	t.Parallel()

	params, err := BuildMessageParams(swarmies.WorkItem{
		TaskID: "swarmies-3oq",
		Title:  "Implement agent registry and A2A dispatch adapter",
		Source: "beads",
	}, swarmies.AgentProfile{ID: swarmies.ProfileGeneralist})
	if err != nil {
		t.Fatalf("BuildMessageParams() error = %v", err)
	}
	if params.Message == nil {
		t.Fatal("params.Message = nil")
	}
	if got := params.Message.ContextID; got != "swarmies-3oq" {
		t.Fatalf("message.ContextID = %q, want %q", got, "swarmies-3oq")
	}
	if got := string(params.Message.TaskID); got != "" {
		t.Fatalf("message.TaskID = %q, want empty for new message/send task", got)
	}
	if params.Config == nil || params.Config.Blocking == nil || !*params.Config.Blocking {
		t.Fatal("params.Config.Blocking = nil/false, want true")
	}

	part, ok := params.Message.Parts[0].(a2acore.TextPart)
	if !ok {
		t.Fatalf("message part type = %T, want TextPart", params.Message.Parts[0])
	}

	var payload agentWorkRequest
	if err := json.Unmarshal([]byte(part.Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.TaskID != "swarmies-3oq" || payload.ContextID != "swarmies-3oq" || payload.Profile != "generalist" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestSummaryAndErrorMessageUseA2AResults(t *testing.T) {
	t.Parallel()

	task := &a2acore.Task{
		Status: a2acore.TaskStatus{
			State: a2acore.TaskStateFailed,
			Message: a2acore.NewMessage(
				a2acore.MessageRoleAgent,
				a2acore.TextPart{Text: `{"task_id":"swarmies-3oq","context_id":"swarmies-3oq","outcome":"blocked","summary":"waiting on credentials","blocked_reason":"missing API key"}`},
			),
		},
	}

	if got := Summary(task); got != "waiting on credentials" {
		t.Fatalf("Summary(task) = %q, want %q", got, "waiting on credentials")
	}
	if got := ErrorMessage(task); got != "missing API key" {
		t.Fatalf("ErrorMessage(task) = %q, want %q", got, "missing API key")
	}
}

func TestRetryMessageUsesStructuredFailureResult(t *testing.T) {
	t.Parallel()

	msg := a2acore.NewMessage(
		a2acore.MessageRoleAgent,
		a2acore.TextPart{Text: `{"task_id":"swarmies-3oq","context_id":"swarmies-3oq","outcome":"failed","summary":"execution failed","error_message":"git apply failed cleanly"}`},
	)

	if got := RetryMessage(msg); got != "Dispatcher marked task for retry after failed outcome: git apply failed cleanly" {
		t.Fatalf("RetryMessage(message) = %q, want structured retry note", got)
	}
}

func TestSummaryUsesStructuredArtifactPayload(t *testing.T) {
	t.Parallel()

	task := &a2acore.Task{
		Status: a2acore.TaskStatus{State: a2acore.TaskStateCompleted},
		Artifacts: []*a2acore.Artifact{
			{
				Parts: []a2acore.Part{
					a2acore.TextPart{Text: `{"task_id":"swarmies-8pk","context_id":"swarmies-8pk","outcome":"success","summary":"claimed over live A2A"}`},
				},
			},
		},
	}

	if got := Summary(task); got != "claimed over live A2A" {
		t.Fatalf("Summary(task) = %q, want %q", got, "claimed over live A2A")
	}
}

func TestExecutionResultPrefersStructuredPayload(t *testing.T) {
	t.Parallel()

	msg := a2acore.NewMessage(
		a2acore.MessageRoleAgent,
		a2acore.TextPart{Text: `{"task_id":"swarmies-9ab","context_id":"swarmies-9ab","outcome":"handoff","summary":"needs coding specialist","handoff":{"target_profile":"coding","reason":"requires implementation work"},"details":{"work_type":"implementation"}}`},
	)

	got, ok := ExecutionResult(msg)
	if !ok {
		t.Fatal("ExecutionResult(message) ok = false, want true")
	}
	if got.Outcome != swarmies.OutcomeHandoff {
		t.Fatalf("ExecutionResult(message).Outcome = %q, want %q", got.Outcome, swarmies.OutcomeHandoff)
	}
	if got.Handoff == nil || got.Handoff.TargetProfile != swarmies.ProfileCoding {
		t.Fatalf("ExecutionResult(message).Handoff = %#v, want coding handoff", got.Handoff)
	}
	if got.Details["work_type"] != "implementation" {
		t.Fatalf("ExecutionResult(message).Details = %#v, want work_type=implementation", got.Details)
	}
}
