package dispatch

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2aclient/agentcard"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/nikstern/swarmies"
	swarmiesa2a "github.com/nikstern/swarmies/internal/a2a"
	"github.com/nikstern/swarmies/internal/registry"
)

type liveBeadsClient struct {
	readyRefs   []swarmies.BeadsTaskRef
	task        swarmies.BeadsTask
	claimed     []string
	closeTaskID string
	closeReason string
	comments    []beadsComment
}

func (f *liveBeadsClient) Ready(context.Context, int) ([]swarmies.BeadsTaskRef, error) {
	return append([]swarmies.BeadsTaskRef(nil), f.readyRefs...), nil
}

func (f *liveBeadsClient) Show(context.Context, string) (swarmies.BeadsTask, error) {
	return f.task, nil
}

func (f *liveBeadsClient) Claim(_ context.Context, id string) error {
	f.claimed = append(f.claimed, id)
	return nil
}

func (f *liveBeadsClient) Close(_ context.Context, id string, reason string) error {
	f.closeTaskID = id
	f.closeReason = reason
	return nil
}

func (f *liveBeadsClient) Comment(_ context.Context, id string, body string) error {
	f.comments = append(f.comments, beadsComment{taskID: id, body: body})
	return nil
}

func TestDispatcherRunOnceAgainstLiveA2AAgent(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	beadsClient := &liveBeadsClient{
		readyRefs: []swarmies.BeadsTaskRef{{ID: "swarmies-8pk"}},
		task: swarmies.BeadsTask{
			ID:          "swarmies-8pk",
			Title:       "Document runnable A2A planner flow for v1 Phase 3",
			Description: "Record the planner-oriented runtime contract and close on success",
		},
	}

	runtime, err := swarmiesa2a.NewRuntime("generalist", beadsClient)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		if err := runtime.RunServer(ctx, swarmiesa2a.ServerConfig{Port: port, PublicURL: baseURL}); err != nil {
			t.Errorf("RunServer() error = %v", err)
		}
	}()

	waitForAgentCard(t, ctx, baseURL)

	dispatcher := NewDispatcher(
		beadsClient,
		registry.NewStatic(swarmies.AgentProfile{
			ID:           swarmies.ProfileGeneralist,
			Name:         "Generalist",
			AgentCardURL: baseURL + a2asrv.WellKnownAgentCardPath,
		}),
		swarmiesa2a.NewGateway(),
		NewDefaultResultPolicy(),
	)

	if err := dispatcher.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if len(beadsClient.claimed) != 1 || beadsClient.claimed[0] != "swarmies-8pk" {
		t.Fatalf("claimed = %#v, want [swarmies-8pk]", beadsClient.claimed)
	}
	if beadsClient.closeTaskID != "swarmies-8pk" {
		t.Fatalf("closed task = %q, want %q", beadsClient.closeTaskID, "swarmies-8pk")
	}
	if beadsClient.closeReason == "" {
		t.Fatal("close reason = empty, want structured summary")
	}
	if len(beadsClient.comments) != 1 {
		t.Fatalf("comments = %#v, want one planner note", beadsClient.comments)
	}
	if beadsClient.comments[0].taskID != "swarmies-8pk" {
		t.Fatalf("comment task = %q, want %q", beadsClient.comments[0].taskID, "swarmies-8pk")
	}
	if beadsClient.comments[0].body == "" {
		t.Fatal("comment body = empty, want planner note")
	}
}

func TestDispatcherRunOnceAgainstLiveA2AAgentKeepsHandoffOpen(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	beadsClient := &liveBeadsClient{
		readyRefs: []swarmies.BeadsTaskRef{{ID: "swarmies-2rt"}},
		task: swarmies.BeadsTask{
			ID:          "swarmies-2rt",
			Title:       "Implement dispatcher retry coverage",
			Description: "Write code to extend the non-success dispatcher behavior",
		},
	}

	runtime, err := swarmiesa2a.NewRuntime("generalist", beadsClient)
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		if err := runtime.RunServer(ctx, swarmiesa2a.ServerConfig{Port: port, PublicURL: baseURL}); err != nil {
			t.Errorf("RunServer() error = %v", err)
		}
	}()

	waitForAgentCard(t, ctx, baseURL)

	dispatcher := NewDispatcher(
		beadsClient,
		registry.NewStatic(swarmies.AgentProfile{
			ID:           swarmies.ProfileGeneralist,
			Name:         "Generalist",
			AgentCardURL: baseURL + a2asrv.WellKnownAgentCardPath,
		}),
		swarmiesa2a.NewGateway(),
		NewDefaultResultPolicy(),
	)

	if err := dispatcher.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if len(beadsClient.claimed) != 1 || beadsClient.claimed[0] != "swarmies-2rt" {
		t.Fatalf("claimed = %#v, want [swarmies-2rt]", beadsClient.claimed)
	}
	if beadsClient.closeTaskID != "" {
		t.Fatalf("closeTaskID = %q, want empty for handoff", beadsClient.closeTaskID)
	}
	if len(beadsClient.comments) != 1 {
		t.Fatalf("comments = %#v, want one planner handoff note", beadsClient.comments)
	}
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func waitForAgentCard(t *testing.T, ctx context.Context, baseURL string) {
	t.Helper()

	var lastErr error
	for range 50 {
		if _, err := agentcard.DefaultResolver.Resolve(ctx, baseURL); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("Resolve(%q) never succeeded: %v", baseURL, lastErr)
}
