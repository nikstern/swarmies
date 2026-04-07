package swarmies

import "testing"

func TestDecodePlannerResult(t *testing.T) {
	t.Parallel()

	result, ok := DecodePlannerResult(`{"task_id":"swarmies-s36","context_id":"swarmies-s36","outcome":"handoff","summary":"requires coding specialist","handoff":{"target_profile":"coding","reason":"implementation work"}}`)
	if !ok {
		t.Fatal("DecodePlannerResult() ok = false, want true")
	}
	if result.Outcome != PlannerOutcomeHandoff {
		t.Fatalf("DecodePlannerResult().Outcome = %q, want %q", result.Outcome, PlannerOutcomeHandoff)
	}
	if result.Handoff == nil || result.Handoff.TargetProfile != ProfileCoding {
		t.Fatalf("DecodePlannerResult().Handoff = %#v, want coding handoff", result.Handoff)
	}
}

func TestDecodePlannerResultRejectsUnknownOutcome(t *testing.T) {
	t.Parallel()

	if _, ok := DecodePlannerResult(`{"outcome":"failed"}`); ok {
		t.Fatal("DecodePlannerResult() ok = true, want false for unknown outcome")
	}
}
