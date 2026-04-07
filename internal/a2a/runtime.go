package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

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

type agentWorkResult struct {
	TaskID       string            `json:"task_id"`
	ContextID    string            `json:"context_id"`
	State        executionState    `json:"state"`
	Summary      string            `json:"summary"`
	Artifacts    []runtimeArtifact `json:"artifacts,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type runtimeArtifact struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type executionState string

const (
	stateSucceeded executionState = "succeeded"
)

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

func (r *Runtime) Run(ctx context.Context, contextID string, content *genai.Content) (agentWorkResult, error) {
	if r == nil || r.runner == nil {
		return agentWorkResult{}, fmt.Errorf("a2a: runtime runner is not configured")
	}

	if contextID == "" {
		contextID = "swarmies"
	}

	var final string
	for event, err := range r.runner.Run(ctx, "dispatcher", contextID, content, agent.RunConfig{}) {
		if err != nil {
			return agentWorkResult{}, fmt.Errorf("a2a: run agent: %w", err)
		}
		if event == nil || event.Content == nil {
			continue
		}
		if text := textFromParts(event.Content.Parts); text != "" {
			final = text
		}
	}

	if final == "" {
		return agentWorkResult{}, fmt.Errorf("a2a: agent returned no final content")
	}

	var result agentWorkResult
	if err := json.Unmarshal([]byte(final), &result); err != nil {
		return agentWorkResult{}, fmt.Errorf("a2a: decode agent result: %w", err)
	}

	return result, nil
}

func newAgent(name string, beads beadsClaimer) (agent.Agent, error) {
	return agent.New(agent.Config{
		Name:        name,
		Description: "Claims one Beads task and returns a structured execution result",
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

				result := agentWorkResult{
					TaskID:    req.TaskID,
					ContextID: req.ContextID,
					State:     stateSucceeded,
					Summary:   fmt.Sprintf("Generalist agent claimed %s and produced a structured result", req.TaskID),
					Artifacts: []runtimeArtifact{
						{
							ID:          "claim-receipt",
							Name:        "beads-claim",
							Description: fmt.Sprintf("Claimed %s for profile %s", req.TaskID, req.Profile),
						},
					},
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
