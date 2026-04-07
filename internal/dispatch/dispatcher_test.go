package dispatch

import (
	"context"
	"reflect"
	"testing"
	"time"

	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/nikstern/swarmies"
)

type fakeBeadsClient struct {
	readyRefs   []swarmies.BeadsTaskRef
	task        swarmies.BeadsTask
	closeTaskID string
	closeReason string
	comments    []beadsComment
	notes       []beadsComment
}

type beadsComment struct {
	taskID string
	body   string
}

func (f *fakeBeadsClient) Ready(context.Context, int) ([]swarmies.BeadsTaskRef, error) {
	return append([]swarmies.BeadsTaskRef(nil), f.readyRefs...), nil
}

func (f *fakeBeadsClient) Show(context.Context, string) (swarmies.BeadsTask, error) {
	return f.task, nil
}

func (f *fakeBeadsClient) Claim(context.Context, string) error {
	return nil
}

func (f *fakeBeadsClient) Close(_ context.Context, id string, reason string) error {
	f.closeTaskID = id
	f.closeReason = reason
	return nil
}

func (f *fakeBeadsClient) Comment(_ context.Context, id string, body string) error {
	f.comments = append(f.comments, beadsComment{taskID: id, body: body})
	return nil
}

func (f *fakeBeadsClient) Note(_ context.Context, id string, body string) error {
	f.notes = append(f.notes, beadsComment{taskID: id, body: body})
	return nil
}

type fakeRegistry struct {
	profile  swarmies.AgentProfile
	selected []swarmies.WorkItem
}

func (f *fakeRegistry) List(context.Context) ([]swarmies.AgentProfile, error) {
	return nil, nil
}

func (f *fakeRegistry) Select(_ context.Context, task swarmies.WorkItem) (swarmies.AgentProfile, error) {
	f.selected = append(f.selected, task)
	return f.profile, nil
}

type fakeGateway struct {
	result  a2acore.SendMessageResult
	profile swarmies.AgentProfile
	params  *a2acore.MessageSendParams
}

func (f *fakeGateway) SendMessage(_ context.Context, profile swarmies.AgentProfile, params *a2acore.MessageSendParams) (a2acore.SendMessageResult, error) {
	f.profile = profile
	f.params = params
	return f.result, nil
}

func TestDispatcherRunOnceHappyPathClosesTask(t *testing.T) {
	t.Parallel()

	discoveredAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	beadsClient := &fakeBeadsClient{
		readyRefs: []swarmies.BeadsTaskRef{{ID: "swarmies-1xm"}},
		task: swarmies.BeadsTask{
			ID:          "swarmies-1xm",
			Title:       "Implement dispatcher RunOnce happy path end to end",
			Description: "Wire Beads through A2A and close on success",
			Labels:      []string{"runtime"},
			RawMetadata: map[string]string{"priority": "P1"},
		},
	}
	registry := &fakeRegistry{
		profile: swarmies.AgentProfile{
			ID:           swarmies.ProfileGeneralist,
			AgentCardURL: "http://127.0.0.1:8080/.well-known/agent-card.json",
		},
	}
	gateway := &fakeGateway{
		result: &a2acore.Task{
			Status: a2acore.TaskStatus{
				State: a2acore.TaskStateCompleted,
				Message: a2acore.NewMessage(
					a2acore.MessageRoleAgent,
					a2acore.TextPart{Text: "structured success summary"},
				),
			},
		},
	}

	dispatcher := NewDispatcher(beadsClient, registry, gateway, NewDefaultResultPolicy())
	dispatcher.now = func() time.Time { return discoveredAt }

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if len(registry.selected) != 1 {
		t.Fatalf("selected count = %d, want 1", len(registry.selected))
	}

	wantItem := swarmies.WorkItem{
		TaskID:       "swarmies-1xm",
		Title:        "Implement dispatcher RunOnce happy path end to end",
		Body:         "Wire Beads through A2A and close on success",
		Labels:       []string{"runtime"},
		Priority:     "P1",
		Source:       "beads",
		DiscoveredAt: discoveredAt,
	}
	if got := registry.selected[0]; !reflect.DeepEqual(got, wantItem) {
		t.Fatalf("selected work item = %#v, want %#v", got, wantItem)
	}

	if gateway.profile.ID != swarmies.ProfileGeneralist {
		t.Fatalf("gateway profile = %q, want %q", gateway.profile.ID, swarmies.ProfileGeneralist)
	}
	if gateway.params == nil || gateway.params.Message == nil {
		t.Fatal("gateway params/message = nil")
	}
	if beadsClient.closeTaskID != "swarmies-1xm" {
		t.Fatalf("closed task = %q, want %q", beadsClient.closeTaskID, "swarmies-1xm")
	}
	if beadsClient.closeReason != "structured success summary" {
		t.Fatalf("close reason = %q, want %q", beadsClient.closeReason, "structured success summary")
	}
	if len(beadsClient.comments) != 0 {
		t.Fatalf("comments = %#v, want none", beadsClient.comments)
	}
}

func TestDispatcherRunOnceFailedTaskLeavesInspectionComment(t *testing.T) {
	t.Parallel()

	beadsClient := &fakeBeadsClient{
		readyRefs: []swarmies.BeadsTaskRef{{ID: "swarmies-1xm"}},
		task: swarmies.BeadsTask{
			ID:          "swarmies-1xm",
			Title:       "Implement dispatcher RunOnce happy path end to end",
			Description: "Wire Beads through A2A and close on success",
		},
	}
	registry := &fakeRegistry{
		profile: swarmies.AgentProfile{
			ID:           swarmies.ProfileGeneralist,
			AgentCardURL: "http://127.0.0.1:8080/.well-known/agent-card.json",
		},
	}
	gateway := &fakeGateway{
		result: &a2acore.Task{
			Status: a2acore.TaskStatus{
				State: a2acore.TaskStateFailed,
				Message: a2acore.NewMessage(
					a2acore.MessageRoleAgent,
					a2acore.TextPart{Text: "tool execution failed"},
				),
			},
		},
	}

	dispatcher := NewDispatcher(beadsClient, registry, gateway, NewDefaultResultPolicy())

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if beadsClient.closeTaskID != "" {
		t.Fatalf("closed task = %q, want none", beadsClient.closeTaskID)
	}

	wantComments := []beadsComment{{
		taskID: "swarmies-1xm",
		body:   "tool execution failed",
	}}
	if !reflect.DeepEqual(beadsClient.comments, wantComments) {
		t.Fatalf("comments = %#v, want %#v", beadsClient.comments, wantComments)
	}
}

func TestDispatcherRunOnceStructuredHandoffLeavesKeepOpenComment(t *testing.T) {
	t.Parallel()

	beadsClient := &fakeBeadsClient{
		readyRefs: []swarmies.BeadsTaskRef{{ID: "swarmies-1xm"}},
		task: swarmies.BeadsTask{
			ID:          "swarmies-1xm",
			Title:       "Investigate dispatcher outcome behavior",
			Description: "Analyze whether this task should route elsewhere",
		},
	}
	registry := &fakeRegistry{
		profile: swarmies.AgentProfile{
			ID:           swarmies.ProfileGeneralist,
			AgentCardURL: "http://127.0.0.1:8080/.well-known/agent-card.json",
		},
	}
	gateway := &fakeGateway{
		result: a2acore.NewMessage(
			a2acore.MessageRoleAgent,
			a2acore.TextPart{Text: `{"task_id":"swarmies-1xm","context_id":"swarmies-1xm","outcome":"handoff","summary":"better suited for coding","handoff":{"target_profile":"coding","reason":"requires code changes"}}`},
		),
	}

	dispatcher := NewDispatcher(beadsClient, registry, gateway, NewDefaultResultPolicy())

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if beadsClient.closeTaskID != "" {
		t.Fatalf("closed task = %q, want none", beadsClient.closeTaskID)
	}

	wantComments := []beadsComment{{
		taskID: "swarmies-1xm",
		body:   "Dispatcher kept task open after handoff outcome: route to coding",
	}}
	if !reflect.DeepEqual(beadsClient.comments, wantComments) {
		t.Fatalf("comments = %#v, want %#v", beadsClient.comments, wantComments)
	}
}

func TestDispatcherRunOnceStructuredFailureLeavesRetryComment(t *testing.T) {
	t.Parallel()

	beadsClient := &fakeBeadsClient{
		readyRefs: []swarmies.BeadsTaskRef{{ID: "swarmies-1xm"}},
		task: swarmies.BeadsTask{
			ID:          "swarmies-1xm",
			Title:       "Repair failed retry behavior",
			Description: "The execution failed and needs another attempt after inspection",
		},
	}
	registry := &fakeRegistry{
		profile: swarmies.AgentProfile{
			ID:           swarmies.ProfileGeneralist,
			AgentCardURL: "http://127.0.0.1:8080/.well-known/agent-card.json",
		},
	}
	gateway := &fakeGateway{
		result: a2acore.NewMessage(
			a2acore.MessageRoleAgent,
			a2acore.TextPart{Text: `{"task_id":"swarmies-1xm","context_id":"swarmies-1xm","outcome":"failed","summary":"execution failed during patch apply","error_message":"git apply failed cleanly"}`},
		),
	}

	dispatcher := NewDispatcher(beadsClient, registry, gateway, NewDefaultResultPolicy())

	if err := dispatcher.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if beadsClient.closeTaskID != "" {
		t.Fatalf("closed task = %q, want none", beadsClient.closeTaskID)
	}

	wantComments := []beadsComment{{
		taskID: "swarmies-1xm",
		body:   "Dispatcher marked task for retry after failed outcome: git apply failed cleanly",
	}}
	if !reflect.DeepEqual(beadsClient.comments, wantComments) {
		t.Fatalf("comments = %#v, want %#v", beadsClient.comments, wantComments)
	}
}
