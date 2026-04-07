package beads

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nikstern/swarmies"
)

func TestClientReadyUsesJSONAndLimit(t *testing.T) {
	t.Parallel()

	client := &Client{
		binary: "bd",
		run: func(_ context.Context, binary string, args ...string) ([]byte, error) {
			if binary != "bd" {
				t.Fatalf("binary = %q, want %q", binary, "bd")
			}

			wantArgs := []string{"ready", "--json", "--limit", "2"}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("args = %v, want %v", args, wantArgs)
			}

			return []byte(`[
				{"id":"swarmies-a"},
				{"id":"swarmies-b"}
			]`), nil
		},
	}

	refs, err := client.Ready(context.Background(), 2)
	if err != nil {
		t.Fatalf("Ready() error = %v", err)
	}

	want := []swarmies.BeadsTaskRef{
		{ID: "swarmies-a"},
		{ID: "swarmies-b"},
	}
	if !reflect.DeepEqual(refs, want) {
		t.Fatalf("Ready() = %#v, want %#v", refs, want)
	}
}

func TestClientShowNormalizesMetadata(t *testing.T) {
	t.Parallel()

	client := &Client{
		binary: "bd",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			wantArgs := []string{"show", "swarmies-cxd", "--json"}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("args = %v, want %v", args, wantArgs)
			}

			return []byte(`[
				{
					"id":"swarmies-cxd",
					"title":"Implement Beads CLI adapter",
					"description":"Thin wrapper around bd",
					"labels":["runtime","cli"],
					"assignee":"Nik Stern",
					"priority":1,
					"metadata":{"profile":"generalist","attempts":2,"ready":true}
				}
			]`), nil
		},
	}

	task, err := client.Show(context.Background(), "swarmies-cxd")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}

	if task.ID != "swarmies-cxd" {
		t.Fatalf("task.ID = %q, want %q", task.ID, "swarmies-cxd")
	}
	if !reflect.DeepEqual(task.Labels, []string{"runtime", "cli"}) {
		t.Fatalf("task.Labels = %#v", task.Labels)
	}

	wantMetadata := map[string]string{
		"profile":  "generalist",
		"attempts": "2",
		"ready":    "true",
		"priority": "P1",
	}
	if !reflect.DeepEqual(task.RawMetadata, wantMetadata) {
		t.Fatalf("task.RawMetadata = %#v, want %#v", task.RawMetadata, wantMetadata)
	}
}

func TestClientShowNotFound(t *testing.T) {
	t.Parallel()

	client := &Client{
		binary: "bd",
		run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`[]`), nil
		},
	}

	_, err := client.Show(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Show() error = %v, want ErrNotFound", err)
	}
}

func TestClientCloseIncludesReason(t *testing.T) {
	t.Parallel()

	client := &Client{
		binary: "bd",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			wantArgs := []string{"close", "swarmies-cxd", "--reason", "completed successfully"}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("args = %v, want %v", args, wantArgs)
			}

			return nil, nil
		},
	}

	if err := client.Close(context.Background(), "swarmies-cxd", "completed successfully"); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClientCommentPassesBody(t *testing.T) {
	t.Parallel()

	client := &Client{
		binary: "bd",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			wantArgs := []string{"comment", "swarmies-cxd", "result summary"}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("args = %v, want %v", args, wantArgs)
			}

			return nil, nil
		},
	}

	if err := client.Comment(context.Background(), "swarmies-cxd", "result summary"); err != nil {
		t.Fatalf("Comment() error = %v", err)
	}
}

func TestClientNotePassesBody(t *testing.T) {
	t.Parallel()

	client := &Client{
		binary: "bd",
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			wantArgs := []string{"note", "swarmies-cxd", "execution report"}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("args = %v, want %v", args, wantArgs)
			}

			return nil, nil
		},
	}

	if err := client.Note(context.Background(), "swarmies-cxd", "execution report"); err != nil {
		t.Fatalf("Note() error = %v", err)
	}
}

func TestCommandErrorIncludesContext(t *testing.T) {
	t.Parallel()

	err := &CommandError{
		Binary:   "bd",
		Args:     []string{"ready", "--json"},
		Stderr:   "boom",
		ExitCode: 1,
		Err:      errors.New("exit status 1"),
	}

	message := err.Error()
	for _, want := range []string{"bd ready --json", "exit=1", "boom"} {
		if !strings.Contains(message, want) {
			t.Fatalf("Error() = %q, want substring %q", message, want)
		}
	}
}
