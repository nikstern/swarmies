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
				a2acore.TextPart{Text: "dispatch failed"},
			),
		},
	}

	if got := Summary(task); got != "dispatch failed" {
		t.Fatalf("Summary(task) = %q, want %q", got, "dispatch failed")
	}
	if got := ErrorMessage(task); got != "dispatch failed" {
		t.Fatalf("ErrorMessage(task) = %q, want %q", got, "dispatch failed")
	}
}

func TestSummaryUsesStructuredArtifactPayload(t *testing.T) {
	t.Parallel()

	task := &a2acore.Task{
		Status: a2acore.TaskStatus{State: a2acore.TaskStateCompleted},
		Artifacts: []*a2acore.Artifact{
			{
				Parts: []a2acore.Part{
					a2acore.TextPart{Text: `{"summary":"claimed over live A2A"}`},
				},
			},
		},
	}

	if got := Summary(task); got != "claimed over live A2A" {
		t.Fatalf("Summary(task) = %q, want %q", got, "claimed over live A2A")
	}
}
