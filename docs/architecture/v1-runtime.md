# v1 Runtime Architecture

## Purpose

This document narrows v1 into a small set of Go interfaces and internal types
that fit the current ADK and A2A model.

Two rules shape the design:

- ADK owns agent runtime concerns such as agent loading and session lifecycle.
- A2A owns the remote execution boundary, agent card, and message or task
  exchange.

v1 should avoid inventing a custom wire protocol between the dispatcher and
agent. Internal Swarmies types can exist, but they should adapt cleanly to ADK
and A2A types. The main exception is one small structured execution result
contract so the runtime and dispatcher can interpret outcomes consistently
across different agent and work types.

## Main Decisions

- The dispatcher is a local control loop.
- Beads remains the source of truth for task state.
- The selected agent claims the Beads task before execution.
- Agents are exposed over A2A, even if v1 starts with a local process.
- v1 agents should always return an A2A task-shaped result for predictability,
  even when the work is short-lived.
- ADK session storage can start with in-memory sessions in v1.
- Swarmies should use ADK `agent.Loader` and `launcher.Config` rather than
  define a parallel runtime bootstrap layer.

## Component Boundaries

### Dispatcher

Owns polling, task normalization, profile selection, dispatch, and outcome
handling.

### Beads Client

Owns all `bd` CLI interaction and maps Beads records into internal task data.

### Agent Registry

Maps a `ProfileID` to a routable A2A agent endpoint plus local metadata used for
selection.

### A2A Gateway

Owns A2A client creation, message construction, request dispatch, and response
normalization.

### ADK Agent Runtime

Owns the actual agent implementation, session service, tool wiring, prompts, and
the ADK launcher configuration used to expose the agent.

## Internal Interfaces

```go
package swarmies

import "context"

type BeadsClient interface {
	Ready(ctx context.Context, limit int) ([]BeadsTaskRef, error)
	Show(ctx context.Context, id string) (BeadsTask, error)
	Claim(ctx context.Context, id string) error
	Close(ctx context.Context, id string, reason string) error
	Comment(ctx context.Context, id string, body string) error
}

type Dispatcher interface {
	RunOnce(ctx context.Context) error
}

type AgentRegistry interface {
	List(ctx context.Context) ([]AgentProfile, error)
	Select(ctx context.Context, task WorkItem) (AgentProfile, error)
}

type A2AGateway interface {
	Dispatch(ctx context.Context, req DispatchRequest) (DispatchResult, error)
}

type ResultPolicy interface {
	Decide(task WorkItem, result DispatchResult) OutcomeDecision
}
```

These interfaces are intentionally local to Swarmies. They should wrap external
libraries rather than replace them.

## Core Types

```go
package swarmies

import "time"

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
	ID               ProfileID
	Name             string
	Description      string
	AgentCardURL     string
	PreferredTransport string
	Skills           []AgentSkill
}

type DispatchRequest struct {
	WorkItem       WorkItem
	Profile        AgentProfile
	TaskID         string
	ContextID      string
	IdempotencyKey string
}

type AgentSkill struct {
	ID          string
	Name        string
	Description string
	Tags        []string
	InputModes  []string
	OutputModes []string
}

type ArtifactRef struct {
	ID          string
	Name        string
	Description string
}

type OutcomeDecision string

const (
	OutcomeClose OutcomeDecision = "close"
	OutcomeKeep  OutcomeDecision = "keep_open"
	OutcomeRetry OutcomeDecision = "retry"
)
```

## Execution Result Contract

v1 agents should return one shared structured payload regardless of whether the
A2A transport surfaces that payload in a task message or artifact. The
dispatcher should depend only on the stable lifecycle fields in this contract.

```go
type ExecutionOutcome string

const (
	OutcomeSuccess    ExecutionOutcome = "success"
	OutcomeBlocked    ExecutionOutcome = "blocked"
	OutcomeNeedsInput ExecutionOutcome = "needs_input"
	OutcomeHandoff    ExecutionOutcome = "handoff"
	OutcomeFailed     ExecutionOutcome = "failed"
)

type ExecutionResult struct {
	TaskID        string
	ContextID     string
	Outcome       ExecutionOutcome
	Summary       string
	Artifacts     []ArtifactRef
	BlockedReason string
	InputRequest  *InputRequest
	Handoff       *HandoffRecommendation
	ErrorMessage  string
	Details       map[string]any
}
```

Dispatcher-facing fields:

- `outcome` is the canonical lifecycle signal
- `task_id` and `context_id` tie the result back to the original dispatch
- `error_message` is reserved for retry-worthy execution failure detail

Agent-facing and human-facing fields:

- `summary` is the short explanation that should appear in close reasons or
  inspection comments
- `blocked_reason`, `input_request`, and `handoff` explain why the planner could
  not proceed directly
- `artifacts` preserves any inspectable receipts or references
- `details` is an extension point for agent-specific payloads that the
  dispatcher can ignore

For v1 the dispatcher should treat `success` as closeable work, `failed` as
retry-worthy execution failure, and `blocked`, `needs_input`, and `handoff` as
keep-open outcomes. Adding a new work type should not require dispatcher
changes unless the system introduces a genuinely new lifecycle outcome.

## Mapping To A2A

Swarmies should model A2A as the external contract:

- `AgentProfile.AgentCardURL` should resolve to an A2A agent card, not just a
  raw endpoint.
- `AgentProfile.PreferredTransport` and `Skills` should mirror fields from
  `a2a.AgentCard`.
- `DispatchRequest` should be converted into `a2a.MessageSendParams`.
- `DispatchResult` should be populated from the returned `a2a.Task` or
  `a2a.Message`, with task-first handling as the default.
- `ContextID` should map directly to the A2A `contextId` used to group related
  interactions.

For v1, Swarmies should treat the A2A response as task-oriented even though A2A
also permits direct message responses.

The concrete A2A client seam in `a2a-go` is:

- agent-card resolution to obtain an `a2a.AgentCard`
- `a2aclient.Factory` to create a compatible `a2aclient.Client`
- `Client.SendMessage(...)` for synchronous dispatch
- optional `Client.SendStreamingMessage(...)` later

The concrete server seam in `a2a-go` is:

- `a2asrv.RequestHandler` for transport-agnostic request handling
- `a2asrv.AgentExecutor` for translating execution into A2A task events

v1 should rely on ADK's existing A2A bridge for the server side instead of
writing a custom `a2asrv.AgentExecutor`.

## Mapping To ADK

ADK should stay behind the agent boundary:

- each agent runs as an ADK-built agent
- each exposed agent is served through the ADK web A2A sublauncher
- `launcher.Config` owns `AgentLoader`, `SessionService`, and A2A options
- `agent.NewSingleLoader(...)` is sufficient for the first agent
- v1 can use `session.InMemoryService()` first and move to persistent sessions
  later

Swarmies should not define its own agent runtime abstraction that duplicates ADK
runner and session responsibilities.

## Happy Path

1. Dispatcher asks Beads for ready work.
2. Dispatcher loads the full task and normalizes it into `WorkItem`.
3. Registry selects an `AgentProfile`.
4. Gateway sends an A2A request to that agent.
5. Agent claims the Beads task before starting execution.
6. Agent completes work and returns a terminal task state.
7. Dispatcher applies `ResultPolicy`.
8. Successful results close the Beads task.

## Deferred From v1

- streaming task updates
- persistent session storage
- push notifications or long-running subscriptions
- multiple concurrent dispatches
- rich artifact storage beyond simple references

## Verified Against Installed Versions

The architecture above was checked against:

- `google.golang.org/adk v1.0.0`
- `github.com/a2aproject/a2a-go v0.3.13`
