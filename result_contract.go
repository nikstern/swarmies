package swarmies

import "encoding/json"

type PlannerOutcome string

const (
	PlannerOutcomeSuccess    PlannerOutcome = "success"
	PlannerOutcomeBlocked    PlannerOutcome = "blocked"
	PlannerOutcomeNeedsInput PlannerOutcome = "needs_input"
	PlannerOutcomeHandoff    PlannerOutcome = "handoff"
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

// PlannerResult is the shared v1 contract for planner-style generalist outcomes.
// Dispatcher-facing fields determine lifecycle behavior; agent-facing fields
// preserve the human-readable context needed for inspection and follow-up.
type PlannerResult struct {
	TaskID        string                 `json:"task_id"`
	ContextID     string                 `json:"context_id"`
	Outcome       PlannerOutcome         `json:"outcome"`
	Summary       string                 `json:"summary"`
	Artifacts     []ArtifactRef          `json:"artifacts,omitempty"`
	BlockedReason string                 `json:"blocked_reason,omitempty"`
	InputRequest  *InputRequest          `json:"input_request,omitempty"`
	Handoff       *HandoffRecommendation `json:"handoff,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

func (r PlannerResult) IsKnownOutcome() bool {
	switch r.Outcome {
	case PlannerOutcomeSuccess, PlannerOutcomeBlocked, PlannerOutcomeNeedsInput, PlannerOutcomeHandoff:
		return true
	default:
		return false
	}
}

func DecodePlannerResult(text string) (PlannerResult, bool) {
	if text == "" {
		return PlannerResult{}, false
	}

	var result PlannerResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return PlannerResult{}, false
	}
	if !result.IsKnownOutcome() {
		return PlannerResult{}, false
	}

	return result, true
}
