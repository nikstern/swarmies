package swarmies

import "testing"

func TestDecodeExecutionResult(t *testing.T) {
	t.Parallel()

	result, ok := DecodeExecutionResult(`{"task_id":"swarmies-mzo","context_id":"swarmies-mzo","outcome":"handoff","summary":"requires coding specialist","handoff":{"target_profile":"coding","reason":"implementation work"},"details":{"work_type":"code_change"}}`)
	if !ok {
		t.Fatal("DecodeExecutionResult() ok = false, want true")
	}
	if result.Outcome != OutcomeHandoff {
		t.Fatalf("DecodeExecutionResult().Outcome = %q, want %q", result.Outcome, OutcomeHandoff)
	}
	if result.Handoff == nil || result.Handoff.TargetProfile != ProfileCoding {
		t.Fatalf("DecodeExecutionResult().Handoff = %#v, want coding handoff", result.Handoff)
	}
	if result.Details["work_type"] != "code_change" {
		t.Fatalf("DecodeExecutionResult().Details = %#v, want work_type=code_change", result.Details)
	}
}

func TestDecodeExecutionResultRejectsUnknownOutcome(t *testing.T) {
	t.Parallel()

	if _, ok := DecodeExecutionResult(`{"outcome":"submitted"}`); ok {
		t.Fatal("DecodeExecutionResult() ok = true, want false for unknown outcome")
	}
}
