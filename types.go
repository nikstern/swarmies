package swarmies

import (
	"context"
	"time"

	a2acore "github.com/a2aproject/a2a-go/a2a"
)

type BeadsClient interface {
	Ready(ctx context.Context, limit int) ([]BeadsTaskRef, error)
	Show(ctx context.Context, id string) (BeadsTask, error)
	Claim(ctx context.Context, id string) error
	Close(ctx context.Context, id string, reason string) error
	Comment(ctx context.Context, id string, body string) error
	Note(ctx context.Context, id string, body string) error
}

type Dispatcher interface {
	RunOnce(ctx context.Context) error
}

type AgentRegistry interface {
	List(ctx context.Context) ([]AgentProfile, error)
	Select(ctx context.Context, task WorkItem) (AgentProfile, error)
}

type A2AGateway interface {
	SendMessage(ctx context.Context, profile AgentProfile, params *a2acore.MessageSendParams) (a2acore.SendMessageResult, error)
}

type ResultPolicy interface {
	Decide(task WorkItem, result a2acore.SendMessageResult) OutcomeDecision
}

type BeadsTaskRef struct {
	ID string
}

type BeadsTask struct {
	ID          string
	Title       string
	Description string
	Labels      []string
	Assignee    string
	RawMetadata map[string]string
}

type WorkItem struct {
	TaskID       string
	Title        string
	Body         string
	Labels       []string
	ProfileHint  string
	Priority     string
	RoutingKey   string
	Source       string
	DiscoveredAt time.Time
}

type ProfileID string

const (
	ProfileGeneralist ProfileID = "generalist"
	ProfileResearch   ProfileID = "research"
	ProfileCoding     ProfileID = "coding"
)

type AgentProfile struct {
	ID           ProfileID
	Name         string
	Description  string
	AgentCardURL string
}

type AgentSkill struct {
	ID          string
	Name        string
	Description string
	Tags        []string
	InputModes  []string
	OutputModes []string
}

type OutcomeDecision string

const (
	OutcomeClose OutcomeDecision = "close"
	OutcomeKeep  OutcomeDecision = "keep_open"
	OutcomeRetry OutcomeDecision = "retry"
)
