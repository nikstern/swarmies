package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"sort"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	launcherweb "google.golang.org/adk/cmd/launcher/web"
	a2alauncher "google.golang.org/adk/cmd/launcher/web/a2a"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/nikstern/swarmies"
)

type beadsClaimer interface {
	Claim(context.Context, string) error
	Comment(context.Context, string, string) error
	Note(context.Context, string, string) error
}

type Runtime struct {
	Name           string
	config         *launcher.Config
	launcher       launcher.SubLauncher
	runner         *runner.Runner
	sessionService session.Service
}

type ServerConfig struct {
	Port      int
	PublicURL string
}

type agentWorkRequest struct {
	TaskID         string            `json:"task_id"`
	ContextID      string            `json:"context_id"`
	Profile        string            `json:"profile"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	WorkItem       swarmies.WorkItem `json:"work_item"`
}

func NewRuntime(name string, beads beadsClaimer) (*Runtime, error) {
	if name == "" {
		name = string(swarmies.ProfileGeneralist)
	}
	if beads == nil {
		return nil, fmt.Errorf("a2a: beads claimer is required")
	}

	rootAgent, err := newAgent(name, beads)
	if err != nil {
		return nil, err
	}

	loader := agent.NewSingleLoader(rootAgent)
	sessionService := session.InMemoryService()
	run, err := runner.New(runner.Config{
		AppName:           rootAgent.Name(),
		Agent:             rootAgent,
		SessionService:    sessionService,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("a2a: create runtime runner: %w", err)
	}

	return &Runtime{
		Name:           name,
		config:         &launcher.Config{AgentLoader: loader, SessionService: sessionService},
		launcher:       launcherweb.NewLauncher(a2alauncher.NewLauncher()),
		runner:         run,
		sessionService: sessionService,
	}, nil
}

func (r *Runtime) RunServer(ctx context.Context, cfg ServerConfig) error {
	if r == nil || r.launcher == nil || r.config == nil {
		return fmt.Errorf("a2a: runtime launcher is not configured")
	}

	port := cfg.Port
	if port <= 0 {
		port = 8080
	}

	publicURL := cfg.PublicURL
	if publicURL == "" {
		publicURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	}

	args := []string{
		"--port", fmt.Sprintf("%d", port),
		"a2a", "--a2a_agent_url", publicURL,
	}
	if _, err := r.launcher.Parse(args); err != nil {
		return fmt.Errorf("a2a: parse launcher args: %w", err)
	}

	return r.launcher.Run(ctx, r.config)
}

func (r *Runtime) Description() string {
	return fmt.Sprintf("ADK-backed runtime for profile %q via the A2A launcher", r.Name)
}

func (r *Runtime) LauncherConfig() *launcher.Config {
	return r.config
}

func (r *Runtime) Launcher() launcher.SubLauncher {
	return r.launcher
}

func (r *Runtime) SessionService() session.Service {
	return r.sessionService
}

func (r *Runtime) Run(ctx context.Context, contextID string, content *genai.Content) (swarmies.ExecutionResult, error) {
	if r == nil || r.runner == nil {
		return swarmies.ExecutionResult{}, fmt.Errorf("a2a: runtime runner is not configured")
	}

	if contextID == "" {
		contextID = "swarmies"
	}

	var final string
	for event, err := range r.runner.Run(ctx, "dispatcher", contextID, content, agent.RunConfig{}) {
		if err != nil {
			return swarmies.ExecutionResult{}, fmt.Errorf("a2a: run agent: %w", err)
		}
		if event == nil || event.Content == nil {
			continue
		}
		if text := textFromParts(event.Content.Parts); text != "" {
			final = text
		}
	}

	if final == "" {
		return swarmies.ExecutionResult{}, fmt.Errorf("a2a: agent returned no final content")
	}

	result, ok := swarmies.DecodeExecutionResult(final)
	if !ok {
		return swarmies.ExecutionResult{}, fmt.Errorf("a2a: decode agent result: unknown execution result contract")
	}

	return result, nil
}

func newAgent(name string, beads beadsClaimer) (agent.Agent, error) {
	return agent.New(agent.Config{
		Name:        name,
		Description: "Claims one Beads task, records Beads execution progress, performs planner-style triage, and returns a structured execution result",
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				req, err := parseAgentWorkRequest(ctx.UserContent())
				if err != nil {
					yield(nil, err)
					return
				}

				if err := beads.Claim(ctx, req.TaskID); err != nil {
					yield(nil, fmt.Errorf("claim task %q: %w", req.TaskID, err))
					return
				}

				result, note := triageWork(req)
				if err := beads.Note(ctx, req.TaskID, executionNote(result)); err != nil {
					yield(nil, fmt.Errorf("append execution note on task %q: %w", req.TaskID, err))
					return
				}
				if err := beads.Comment(ctx, req.TaskID, note); err != nil {
					yield(nil, fmt.Errorf("comment on task %q: %w", req.TaskID, err))
					return
				}

				payload, err := json.Marshal(result)
				if err != nil {
					yield(nil, fmt.Errorf("marshal result for %q: %w", req.TaskID, err))
					return
				}

				event := session.NewEvent(ctx.InvocationID())
				event.Content = &genai.Content{
					Role: genai.RoleModel,
					Parts: []*genai.Part{
						{Text: string(payload)},
					},
				}
				yield(event, nil)
			}
		},
	})
}

func triageWork(req agentWorkRequest) (swarmies.ExecutionResult, string) {
	result := swarmies.ExecutionResult{
		TaskID:    req.TaskID,
		ContextID: req.ContextID,
		Artifacts: []swarmies.ArtifactRef{
			{
				ID:          "claim-receipt",
				Name:        "beads-claim",
				Description: fmt.Sprintf("Claimed %s for profile %s", req.TaskID, req.Profile),
			},
		},
		Details: map[string]any{
			"agent_type": "generalist",
			"work_type":  "triage",
		},
	}

	text := strings.ToLower(strings.Join([]string{
		req.WorkItem.Title,
		req.WorkItem.Body,
		strings.Join(req.WorkItem.Labels, " "),
	}, "\n"))

	switch {
	case containsAny(text, "fail", "failure", "error", "broken", "exception"):
		result.Outcome = swarmies.OutcomeFailed
		result.Summary = fmt.Sprintf("Generalist planner could not execute %s cleanly", req.TaskID)
		result.ErrorMessage = "Task description indicates an execution failure that should be retried after inspection."
		result.Details["triage_outcome"] = "failed"
		result.Details["recommended_next_step"] = "Inspect the failure details, fix the underlying issue, and redispatch."
	case containsAny(text, "blocked by", "waiting on", "awaiting", "depends on", "dependency"):
		result.Outcome = swarmies.OutcomeBlocked
		result.Summary = fmt.Sprintf("Generalist planner marked %s as blocked", req.TaskID)
		result.BlockedReason = "Task description indicates an external dependency or waiting condition."
		result.Details["triage_outcome"] = "blocked"
		result.Details["recommended_next_step"] = "Resolve the dependency before redispatch."
	case containsAny(text, "needs input", "need input", "clarify", "clarification", "open question", "decision", "which ", "what "):
		result.Outcome = swarmies.OutcomeNeedsInput
		result.Summary = fmt.Sprintf("Generalist planner needs more input before proceeding on %s", req.TaskID)
		result.InputRequest = &swarmies.InputRequest{
			Question: "What missing decision or clarification should guide this task?",
			Details:  "The task description reads as ambiguous or decision-dependent for v1 triage.",
		}
		result.Details["triage_outcome"] = "needs_input"
		result.Details["recommended_next_step"] = "Add the missing product or technical guidance to the bead."
	case containsAny(text, "research", "investigate", "analyze", "analysis", "evaluate", "compare", "survey"):
		result.Outcome = swarmies.OutcomeHandoff
		result.Summary = fmt.Sprintf("Generalist planner recommends handing %s to the research profile", req.TaskID)
		result.Handoff = &swarmies.HandoffRecommendation{
			TargetProfile: swarmies.ProfileResearch,
			Reason:        "The task reads like analysis or information-gathering work.",
		}
		result.Details["triage_outcome"] = "handoff"
		result.Details["recommended_next_step"] = "Route this bead to the research specialist."
	case containsAny(text, "implement", "implementation", "build", "write code", "refactor", "test", "fix"):
		result.Outcome = swarmies.OutcomeHandoff
		result.Summary = fmt.Sprintf("Generalist planner recommends handing %s to the coding profile", req.TaskID)
		result.Handoff = &swarmies.HandoffRecommendation{
			TargetProfile: swarmies.ProfileCoding,
			Reason:        "The task requires implementation-oriented work rather than generalist triage.",
		}
		result.Details["triage_outcome"] = "handoff"
		result.Details["recommended_next_step"] = "Route this bead to the coding specialist."
	default:
		result.Outcome = swarmies.OutcomeSuccess
		result.Summary = fmt.Sprintf("Generalist planner triaged %s as actionable and recorded a plan", req.TaskID)
		result.Artifacts = append(result.Artifacts, swarmies.ArtifactRef{
			ID:          "triage-note",
			Name:        "planner-note",
			Description: fmt.Sprintf("Planner note recorded for %s", req.TaskID),
		})
		result.Details["triage_outcome"] = "actionable"
		result.Details["recommended_next_step"] = "Proceed with the next bounded planning step."
	}

	return result, inspectionNote(result)
}

func inspectionNote(result swarmies.ExecutionResult) string {
	lines := []string{
		fmt.Sprintf("Planner triage: %s", result.Outcome),
		fmt.Sprintf("Summary: %s", result.Summary),
	}

	switch result.Outcome {
	case swarmies.OutcomeFailed:
		lines = append(lines, fmt.Sprintf("Failure detail: %s", result.ErrorMessage))
	case swarmies.OutcomeBlocked:
		lines = append(lines, fmt.Sprintf("Blocked reason: %s", result.BlockedReason))
	case swarmies.OutcomeNeedsInput:
		if result.InputRequest != nil {
			lines = append(lines, fmt.Sprintf("Input request: %s", result.InputRequest.Question))
			if result.InputRequest.Details != "" {
				lines = append(lines, fmt.Sprintf("Details: %s", result.InputRequest.Details))
			}
		}
	case swarmies.OutcomeHandoff:
		if result.Handoff != nil {
			lines = append(lines, fmt.Sprintf("Recommended profile: %s", result.Handoff.TargetProfile))
			if result.Handoff.Reason != "" {
				lines = append(lines, fmt.Sprintf("Reason: %s", result.Handoff.Reason))
			}
		}
	case swarmies.OutcomeSuccess:
		if next, _ := result.Details["recommended_next_step"].(string); next != "" {
			lines = append(lines, fmt.Sprintf("Next step: %s", next))
		}
	}

	return strings.Join(lines, "\n")
}

func executionNote(result swarmies.ExecutionResult) string {
	record := swarmies.BeadsProgressRecord{
		Phase:   "agent_execution",
		Status:  result.Outcome,
		Summary: result.Summary,
		Details: executionDetails(result),
	}

	lines := []string{
		"[swarmies/execution]",
		fmt.Sprintf("phase: %s", record.Phase),
		fmt.Sprintf("status: %s", record.Status),
		fmt.Sprintf("summary: %s", record.Summary),
	}

	if len(record.Details) == 0 {
		return strings.Join(lines, "\n")
	}

	keys := make([]string, 0, len(record.Details))
	for key := range record.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", key, record.Details[key]))
	}

	return strings.Join(lines, "\n")
}

func executionDetails(result swarmies.ExecutionResult) map[string]string {
	details := map[string]string{}

	if result.TaskID != "" {
		details["task_id"] = result.TaskID
	}
	if result.ContextID != "" {
		details["context_id"] = result.ContextID
	}
	if result.BlockedReason != "" {
		details["blocked_reason"] = result.BlockedReason
	}
	if result.ErrorMessage != "" {
		details["error_message"] = result.ErrorMessage
	}
	if result.InputRequest != nil {
		if result.InputRequest.Question != "" {
			details["input_question"] = result.InputRequest.Question
		}
		if result.InputRequest.Details != "" {
			details["input_details"] = result.InputRequest.Details
		}
	}
	if result.Handoff != nil {
		if result.Handoff.TargetProfile != "" {
			details["handoff_target"] = string(result.Handoff.TargetProfile)
		}
		if result.Handoff.Reason != "" {
			details["handoff_reason"] = result.Handoff.Reason
		}
	}
	if next, _ := result.Details["recommended_next_step"].(string); next != "" {
		details["recommended_next_step"] = next
	}
	if outcome, _ := result.Details["triage_outcome"].(string); outcome != "" {
		details["triage_outcome"] = outcome
	}

	return details
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func parseAgentWorkRequest(content *genai.Content) (agentWorkRequest, error) {
	if content == nil {
		return agentWorkRequest{}, fmt.Errorf("a2a: missing user content")
	}

	body := textFromParts(content.Parts)
	if body == "" {
		return agentWorkRequest{}, fmt.Errorf("a2a: missing request payload")
	}

	var req agentWorkRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		return agentWorkRequest{}, fmt.Errorf("a2a: decode work request: %w", err)
	}
	if req.TaskID == "" {
		return agentWorkRequest{}, fmt.Errorf("a2a: work request is missing task_id")
	}
	if req.ContextID == "" {
		req.ContextID = req.TaskID
	}
	return req, nil
}

func textFromParts(parts []*genai.Part) string {
	for _, part := range parts {
		if part != nil && part.Text != "" {
			return part.Text
		}
	}
	return ""
}
