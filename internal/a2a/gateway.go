package a2a

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/a2aproject/a2a-go/a2aclient/agentcard"
	"github.com/nikstern/swarmies"
)

type cardResolver interface {
	Resolve(context.Context, string, ...agentcard.ResolveOption) (*a2acore.AgentCard, error)
}

type messageSender interface {
	SendMessage(context.Context, *a2acore.MessageSendParams) (a2acore.SendMessageResult, error)
}

type Gateway struct {
	resolver     cardResolver
	createClient func(context.Context, *a2acore.AgentCard) (messageSender, error)
}

func NewGateway() *Gateway {
	factory := a2aclient.NewFactory()

	return &Gateway{
		resolver: agentcard.DefaultResolver,
		createClient: func(ctx context.Context, card *a2acore.AgentCard) (messageSender, error) {
			return factory.CreateFromCard(ctx, card)
		},
	}
}

func (g *Gateway) SendMessage(ctx context.Context, profile swarmies.AgentProfile, params *a2acore.MessageSendParams) (a2acore.SendMessageResult, error) {
	if g == nil || g.resolver == nil || g.createClient == nil {
		return nil, fmt.Errorf("a2a: gateway is not configured")
	}
	if profile.AgentCardURL == "" {
		return nil, fmt.Errorf("a2a: profile %q is missing an agent card URL", profile.ID)
	}
	if params == nil || params.Message == nil {
		return nil, fmt.Errorf("a2a: message params are required")
	}

	card, err := g.resolveCard(ctx, profile.AgentCardURL)
	if err != nil {
		return nil, fmt.Errorf("a2a: resolve agent card for %q: %w", profile.ID, err)
	}

	client, err := g.createClient(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("a2a: create client for %q: %w", profile.ID, err)
	}

	result, err := client.SendMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("a2a: send message for %q: %w", profile.ID, err)
	}

	return result, nil
}

func (g *Gateway) resolveCard(ctx context.Context, raw string) (*a2acore.AgentCard, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse agent card url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("agent card url must be absolute: %q", raw)
	}

	base := (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
	path := strings.TrimSpace(parsed.EscapedPath())
	if path == "" || path == "/" {
		return g.resolver.Resolve(ctx, base)
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}

	return g.resolver.Resolve(ctx, base, agentcard.WithPath(path))
}
