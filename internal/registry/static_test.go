package registry

import (
	"context"
	"testing"

	"github.com/nikstern/swarmies"
)

func TestDefaultProfilesIncludeGeneralist(t *testing.T) {
	t.Parallel()

	registry := NewStatic(DefaultProfiles()...)

	profile, err := registry.Select(context.Background(), swarmies.WorkItem{})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}

	if profile.ID != swarmies.ProfileGeneralist {
		t.Fatalf("profile.ID = %q, want %q", profile.ID, swarmies.ProfileGeneralist)
	}
	if profile.AgentCardURL == "" {
		t.Fatal("profile.AgentCardURL = empty")
	}
}

func TestStaticRegistryUsesProfileHintBeforeLabels(t *testing.T) {
	t.Parallel()

	registry := NewStatic(DefaultProfiles()...)

	profile, err := registry.Select(context.Background(), swarmies.WorkItem{
		ProfileHint: "coding",
		Labels:      []string{"research"},
	})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}

	if profile.ID != swarmies.ProfileCoding {
		t.Fatalf("profile.ID = %q, want %q", profile.ID, swarmies.ProfileCoding)
	}
}
