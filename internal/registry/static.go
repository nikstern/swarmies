package registry

import (
	"context"
	"errors"
	"strings"

	"github.com/nikstern/swarmies"
)

var ErrNoProfiles = errors.New("registry: no profiles configured")

type StaticRegistry struct {
	profiles []swarmies.AgentProfile
}

func NewStatic(profiles ...swarmies.AgentProfile) *StaticRegistry {
	return &StaticRegistry{profiles: append([]swarmies.AgentProfile(nil), profiles...)}
}

func DefaultProfiles() []swarmies.AgentProfile {
	return []swarmies.AgentProfile{
		{
			ID:                 swarmies.ProfileGeneralist,
			Name:               "Generalist",
			Description:        "Default profile for uncategorized work",
			AgentCardURL:       "http://127.0.0.1:8080/.well-known/agent-card.json",
			PreferredTransport: "a2a-http",
			Skills: []swarmies.AgentSkill{
				{
					ID:          "beads-claim-and-report",
					Name:        "Claim Beads work",
					Description: "Claims a Beads issue and returns a structured execution result",
					Tags:        []string{"beads", "dispatch", "generalist"},
					InputModes:  []string{"text/plain"},
					OutputModes: []string{"application/json", "text/plain"},
				},
			},
		},
		{
			ID:                 swarmies.ProfileResearch,
			Name:               "Research",
			Description:        "Profile for analysis and information gathering",
			AgentCardURL:       "http://127.0.0.1:8081/.well-known/agent-card.json",
			PreferredTransport: "a2a-http",
		},
		{
			ID:                 swarmies.ProfileCoding,
			Name:               "Coding",
			Description:        "Profile for implementation-oriented work",
			AgentCardURL:       "http://127.0.0.1:8082/.well-known/agent-card.json",
			PreferredTransport: "a2a-http",
		},
	}
}

func (r *StaticRegistry) List(context.Context) ([]swarmies.AgentProfile, error) {
	if len(r.profiles) == 0 {
		return nil, ErrNoProfiles
	}

	return append([]swarmies.AgentProfile(nil), r.profiles...), nil
}

func (r *StaticRegistry) Select(_ context.Context, task swarmies.WorkItem) (swarmies.AgentProfile, error) {
	if len(r.profiles) == 0 {
		return swarmies.AgentProfile{}, ErrNoProfiles
	}

	if task.ProfileHint != "" {
		for _, profile := range r.profiles {
			if strings.EqualFold(string(profile.ID), task.ProfileHint) {
				return profile, nil
			}
		}
	}

	for _, label := range task.Labels {
		switch strings.ToLower(label) {
		case "research":
			if profile, ok := r.byID(swarmies.ProfileResearch); ok {
				return profile, nil
			}
		case "coding":
			if profile, ok := r.byID(swarmies.ProfileCoding); ok {
				return profile, nil
			}
		}
	}

	if profile, ok := r.byID(swarmies.ProfileGeneralist); ok {
		return profile, nil
	}

	return r.profiles[0], nil
}

func (r *StaticRegistry) byID(id swarmies.ProfileID) (swarmies.AgentProfile, bool) {
	for _, profile := range r.profiles {
		if profile.ID == id {
			return profile, true
		}
	}

	return swarmies.AgentProfile{}, false
}
