package dispatch

import (
	"encoding/json"
	"fmt"
	"strings"

	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/nikstern/swarmies"
)

type agentWorkRequest struct {
	TaskID         string            `json:"task_id"`
	ContextID      string            `json:"context_id"`
	Profile        string            `json:"profile"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	WorkItem       swarmies.WorkItem `json:"work_item"`
}

func BuildMessageParams(workItem swarmies.WorkItem, profile swarmies.AgentProfile) (*a2acore.MessageSendParams, error) {
	blocking := true
	payload, err := json.Marshal(agentWorkRequest{
		TaskID:         workItem.TaskID,
		ContextID:      workItem.TaskID,
		Profile:        string(profile.ID),
		IdempotencyKey: workItem.TaskID,
		WorkItem:       workItem,
	})
	if err != nil {
		return nil, fmt.Errorf("encode work item: %w", err)
	}

	msg := a2acore.NewMessage(a2acore.MessageRoleUser, a2acore.TextPart{Text: string(payload)})
	msg.ContextID = workItem.TaskID

	return &a2acore.MessageSendParams{
		Config: &a2acore.MessageSendConfig{
			Blocking:            &blocking,
			AcceptedOutputModes: []string{"application/json", "text/plain"},
		},
		Message: msg,
		Metadata: map[string]any{
			"source":          workItem.Source,
			"profile":         string(profile.ID),
			"idempotency_key": workItem.TaskID,
		},
	}, nil
}

func Summary(result a2acore.SendMessageResult) string {
	if structured, ok := PlannerResult(result); ok {
		return structured.Summary
	}

	switch typed := result.(type) {
	case *a2acore.Task:
		return firstNonEmpty(messageText(typed.Status.Message), taskArtifactsText(typed))
	case *a2acore.Message:
		return messageText(typed)
	default:
		return ""
	}
}

func ErrorMessage(result a2acore.SendMessageResult) string {
	if structured, ok := PlannerResult(result); ok {
		return firstNonEmpty(structured.ErrorMessage, structured.BlockedReason, structured.Summary)
	}

	switch typed := result.(type) {
	case *a2acore.Task:
		if typed.Status.State == a2acore.TaskStateFailed || typed.Status.State == a2acore.TaskStateCanceled || typed.Status.State == a2acore.TaskStateRejected {
			return firstNonEmpty(messageText(typed.Status.Message), taskArtifactsText(typed))
		}
	case *a2acore.Message:
		return messageText(typed)
	}
	return ""
}

func PlannerResult(result a2acore.SendMessageResult) (swarmies.PlannerResult, bool) {
	switch typed := result.(type) {
	case *a2acore.Task:
		return firstPlannerResult(messageText(typed.Status.Message), taskArtifactsText(typed))
	case *a2acore.Message:
		return firstPlannerResult(messageText(typed))
	default:
		return swarmies.PlannerResult{}, false
	}
}

func messageText(msg *a2acore.Message) string {
	if msg == nil {
		return ""
	}

	parts := make([]string, 0, len(msg.Parts))
	for _, part := range msg.Parts {
		switch typed := part.(type) {
		case a2acore.TextPart:
			if typed.Text != "" {
				parts = append(parts, typed.Text)
			}
		case a2acore.DataPart:
			raw, err := json.Marshal(typed.Data)
			if err == nil && len(raw) > 0 {
				parts = append(parts, string(raw))
			}
		}
	}

	return strings.Join(parts, "\n")
}

func taskArtifactsText(task *a2acore.Task) string {
	if task == nil {
		return ""
	}

	parts := make([]string, 0, len(task.Artifacts))
	for _, artifact := range task.Artifacts {
		for _, part := range artifact.Parts {
			switch typed := part.(type) {
			case a2acore.TextPart:
				if typed.Text != "" {
					parts = append(parts, typed.Text)
				}
			case a2acore.DataPart:
				raw, err := json.Marshal(typed.Data)
				if err == nil && len(raw) > 0 {
					parts = append(parts, string(raw))
				}
			}
		}
	}

	return strings.Join(parts, "\n")
}

func firstPlannerResult(values ...string) (swarmies.PlannerResult, bool) {
	for _, value := range values {
		if result, ok := swarmies.DecodePlannerResult(value); ok {
			return result, true
		}
	}
	return swarmies.PlannerResult{}, false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
