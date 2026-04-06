package dispatch

import (
	"testing"

	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/nikstern/swarmies"
)

func TestDefaultResultPolicyDecide(t *testing.T) {
	t.Parallel()

	policy := NewDefaultResultPolicy()

	tests := []struct {
		name   string
		result a2acore.SendMessageResult
		want   swarmies.OutcomeDecision
	}{
		{
			name:   "message closes",
			result: a2acore.NewMessage(a2acore.MessageRoleAgent, a2acore.TextPart{Text: "done"}),
			want:   swarmies.OutcomeClose,
		},
		{
			name: "completed task closes",
			result: &a2acore.Task{
				Status: a2acore.TaskStatus{State: a2acore.TaskStateCompleted},
			},
			want: swarmies.OutcomeClose,
		},
		{
			name: "failed task retries",
			result: &a2acore.Task{
				Status: a2acore.TaskStatus{State: a2acore.TaskStateFailed},
			},
			want: swarmies.OutcomeRetry,
		},
		{
			name: "working task stays open",
			result: &a2acore.Task{
				Status: a2acore.TaskStatus{State: a2acore.TaskStateWorking},
			},
			want: swarmies.OutcomeKeep,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := policy.Decide(swarmies.WorkItem{}, tt.result); got != tt.want {
				t.Fatalf("Decide() = %q, want %q", got, tt.want)
			}
		})
	}
}
