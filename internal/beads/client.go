package beads

import (
	"context"

	"github.com/nikstern/swarmies"
)

const DefaultBinary = "bd"

type Client struct {
	binary string
}

func NewClient(binary string) *Client {
	if binary == "" {
		binary = DefaultBinary
	}

	return &Client{binary: binary}
}

func (c *Client) Ready(context.Context, int) ([]swarmies.BeadsTaskRef, error) {
	return nil, nil
}

func (c *Client) Show(context.Context, string) (swarmies.BeadsTask, error) {
	return swarmies.BeadsTask{}, nil
}

func (c *Client) Claim(context.Context, string) error {
	return nil
}

func (c *Client) Close(context.Context, string, string) error {
	return nil
}

func (c *Client) Comment(context.Context, string, string) error {
	return nil
}
