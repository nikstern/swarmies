package swarmies

import "encoding/json"

type ExecutionOutcome string

const (
	OutcomeSuccess    ExecutionOutcome = "success"
	OutcomeBlocked    ExecutionOutcome = "blocked"
	OutcomeNeedsInput ExecutionOutcome = "needs_input"
	OutcomeHandoff    ExecutionOutcome = "handoff"
	OutcomeFailed     ExecutionOutcome = "failed"
)

type ArtifactRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type InputRequest struct {
	Question string `json:"question"`
	Details  string `json:"details,omitempty"`
}

type HandoffRecommendation struct {
	TargetProfile ProfileID `json:"target_profile,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

// ExecutionResult is the shared v1 result envelope across agent and work types.
// Dispatcher-facing lifecycle fields stay stable while agents can attach
// additional typed context in Details without changing dispatcher parsing.
type ExecutionResult struct {
	TaskID        string                 `json:"task_id"`
	ContextID     string                 `json:"context_id"`
	Outcome       ExecutionOutcome       `json:"outcome"`
	Summary       string                 `json:"summary"`
	Artifacts     []ArtifactRef          `json:"artifacts,omitempty"`
	BlockedReason string                 `json:"blocked_reason,omitempty"`
	InputRequest  *InputRequest          `json:"input_request,omitempty"`
	Handoff       *HandoffRecommendation `json:"handoff,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Details       map[string]any         `json:"details,omitempty"`
}

func (r ExecutionResult) IsKnownOutcome() bool {
	switch r.Outcome {
	case OutcomeSuccess, OutcomeBlocked, OutcomeNeedsInput, OutcomeHandoff, OutcomeFailed:
		return true
	default:
		return false
	}
}

func DecodeExecutionResult(text string) (ExecutionResult, bool) {
	if text == "" {
		return ExecutionResult{}, false
	}

	var result ExecutionResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return ExecutionResult{}, false
	}
	if !result.IsKnownOutcome() {
		return ExecutionResult{}, false
	}

	return result, true
}
