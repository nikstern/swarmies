package a2a

import (
	"context"
	"testing"

	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient/agentcard"
	"github.com/nikstern/swarmies"
)

type fakeClient struct {
	params *a2acore.MessageSendParams
	result a2acore.SendMessageResult
	err    error
}

func (f *fakeClient) SendMessage(_ context.Context, params *a2acore.MessageSendParams) (a2acore.SendMessageResult, error) {
	f.params = params
	return f.result, f.err
}

func TestGatewaySendMessageUsesResolvedCardAndPassesParamsThrough(t *testing.T) {
	t.Parallel()

	card := &a2acore.AgentCard{
		Name:               "Generalist",
		URL:                "http://127.0.0.1:8080",
		PreferredTransport: a2acore.TransportProtocolJSONRPC,
		ProtocolVersion:    string(a2acore.Version),
	}
	resolver := &gatewayResolver{card: card}
	client := &fakeClient{
		result: a2acore.NewMessage(a2acore.MessageRoleAgent, a2acore.TextPart{Text: "ok"}),
	}

	gateway := &Gateway{
		resolver: resolver,
		createClient: func(context.Context, *a2acore.AgentCard) (messageSender, error) {
			return client, nil
		},
	}

	params := &a2acore.MessageSendParams{
		Message: a2acore.NewMessage(a2acore.MessageRoleUser, a2acore.TextPart{Text: "hello"}),
	}
	result, err := gateway.SendMessage(context.Background(), swarmies.AgentProfile{
		ID:           swarmies.ProfileGeneralist,
		AgentCardURL: "http://127.0.0.1:8080/.well-known/agent-card.json",
	}, params)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if resolver.base != "http://127.0.0.1:8080" {
		t.Fatalf("resolver base = %q, want %q", resolver.base, "http://127.0.0.1:8080")
	}
	if client.params != params {
		t.Fatal("gateway changed message params before sending")
	}
	if result != client.result {
		t.Fatal("gateway changed send result")
	}
}

func TestGatewaySendMessageRejectsMissingProfileURL(t *testing.T) {
	t.Parallel()

	_, err := NewGateway().SendMessage(context.Background(), swarmies.AgentProfile{}, &a2acore.MessageSendParams{
		Message: a2acore.NewMessage(a2acore.MessageRoleUser, a2acore.TextPart{Text: "hello"}),
	})
	if err == nil {
		t.Fatal("SendMessage() error = nil, want error")
	}
}

type gatewayResolver struct {
	base string
	card *a2acore.AgentCard
}

func (r *gatewayResolver) Resolve(_ context.Context, base string, _ ...agentcard.ResolveOption) (*a2acore.AgentCard, error) {
	r.base = base
	return r.card, nil
}
