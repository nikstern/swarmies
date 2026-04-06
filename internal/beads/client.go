package beads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nikstern/swarmies"
)

const DefaultBinary = "bd"

var ErrNotFound = errors.New("beads: task not found")

type Client struct {
	binary string
	run    runner
}

func NewClient(binary string) *Client {
	if binary == "" {
		binary = DefaultBinary
	}

	return &Client{
		binary: binary,
		run:    execRunner,
	}
}

func (c *Client) Ready(ctx context.Context, limit int) ([]swarmies.BeadsTaskRef, error) {
	args := []string{"ready", "--json"}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}

	issues, err := c.readIssues(ctx, args...)
	if err != nil {
		return nil, err
	}

	refs := make([]swarmies.BeadsTaskRef, 0, len(issues))
	for _, issue := range issues {
		if issue.ID == "" {
			continue
		}

		refs = append(refs, swarmies.BeadsTaskRef{ID: issue.ID})
	}

	return refs, nil
}

func (c *Client) Show(ctx context.Context, id string) (swarmies.BeadsTask, error) {
	issues, err := c.readIssues(ctx, "show", id, "--json")
	if err != nil {
		return swarmies.BeadsTask{}, err
	}
	if len(issues) == 0 {
		return swarmies.BeadsTask{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	return issues[0].toTask(), nil
}

func (c *Client) Claim(ctx context.Context, id string) error {
	_, err := c.run(ctx, c.binary, "update", id, "--claim")
	return err
}

func (c *Client) Close(ctx context.Context, id string, reason string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}

	_, err := c.run(ctx, c.binary, args...)
	return err
}

func (c *Client) Comment(ctx context.Context, id string, body string) error {
	_, err := c.run(ctx, c.binary, "comment", id, body)
	return err
}

func (c *Client) readIssues(ctx context.Context, args ...string) ([]issueRecord, error) {
	output, err := c.run(ctx, c.binary, args...)
	if err != nil {
		return nil, err
	}

	var issues []issueRecord
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("beads: decode %q output: %w", strings.Join(args, " "), err)
	}

	return issues, nil
}

type runner func(ctx context.Context, binary string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, binary string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &CommandError{
			Binary:   binary,
			Args:     append([]string(nil), args...),
			Stderr:   strings.TrimSpace(stderr.String()),
			ExitCode: exitCode(err),
			Err:      err,
		}
	}

	return stdout.Bytes(), nil
}

type CommandError struct {
	Binary   string
	Args     []string
	Stderr   string
	ExitCode int
	Err      error
}

func (e *CommandError) Error() string {
	parts := []string{fmt.Sprintf("beads: command failed: %s %s", e.Binary, strings.Join(e.Args, " "))}
	if e.ExitCode >= 0 {
		parts = append(parts, fmt.Sprintf("exit=%d", e.ExitCode))
	}
	if e.Stderr != "" {
		parts = append(parts, e.Stderr)
	}

	return strings.Join(parts, ": ")
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

func exitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}

type issueRecord struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Labels      []string       `json:"labels"`
	Assignee    string         `json:"assignee"`
	Priority    int            `json:"priority"`
	Metadata    map[string]any `json:"metadata"`
}

func (r issueRecord) toTask() swarmies.BeadsTask {
	return swarmies.BeadsTask{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Labels:      append([]string(nil), r.Labels...),
		Assignee:    r.Assignee,
		RawMetadata: r.rawMetadata(),
	}
}

func (r issueRecord) rawMetadata() map[string]string {
	metadata := make(map[string]string, len(r.Metadata)+1)
	for key, value := range r.Metadata {
		switch typed := value.(type) {
		case string:
			metadata[key] = typed
		case float64:
			metadata[key] = strconv.FormatFloat(typed, 'f', -1, 64)
		case bool:
			metadata[key] = strconv.FormatBool(typed)
		default:
			raw, err := json.Marshal(typed)
			if err != nil {
				continue
			}
			metadata[key] = string(raw)
		}
	}

	if r.Priority > 0 {
		metadata["priority"] = fmt.Sprintf("P%d", r.Priority)
	}

	if len(metadata) == 0 {
		return nil
	}

	return metadata
}
