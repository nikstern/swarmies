package dispatch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikstern/swarmies"
)

const defaultReadyLimit = 1

var ErrMissingDependency = errors.New("dispatch: missing dependency")

type Dispatcher struct {
	beads    swarmies.BeadsClient
	registry swarmies.AgentRegistry
	gateway  swarmies.A2AGateway
	policy   swarmies.ResultPolicy
	now      func() time.Time
}

func NewDispatcher(
	beads swarmies.BeadsClient,
	registry swarmies.AgentRegistry,
	gateway swarmies.A2AGateway,
	policy swarmies.ResultPolicy,
) *Dispatcher {
	return &Dispatcher{
		beads:    beads,
		registry: registry,
		gateway:  gateway,
		policy:   policy,
		now:      time.Now,
	}
}

func (d *Dispatcher) RunOnce(ctx context.Context) error {
	if d.beads == nil || d.registry == nil || d.gateway == nil || d.policy == nil {
		return ErrMissingDependency
	}

	refs, err := d.beads.Ready(ctx, defaultReadyLimit)
	if err != nil {
		return fmt.Errorf("list ready tasks: %w", err)
	}
	if len(refs) == 0 {
		return nil
	}

	task, err := d.beads.Show(ctx, refs[0].ID)
	if err != nil {
		return fmt.Errorf("show task %q: %w", refs[0].ID, err)
	}

	workItem := d.normalize(task)

	profile, err := d.registry.Select(ctx, workItem)
	if err != nil {
		return fmt.Errorf("select profile for %q: %w", workItem.TaskID, err)
	}

	params, err := BuildMessageParams(workItem, profile)
	if err != nil {
		return fmt.Errorf("build A2A request for %q: %w", workItem.TaskID, err)
	}

	result, err := d.gateway.SendMessage(ctx, profile, params)
	if err != nil {
		return fmt.Errorf("dispatch task %q: %w", workItem.TaskID, err)
	}

	switch d.policy.Decide(workItem, result) {
	case swarmies.OutcomeClose:
		return d.beads.Close(ctx, workItem.TaskID, Summary(result))
	case swarmies.OutcomeKeep:
		if note := KeepOpenMessage(result); note != "" {
			return d.beads.Comment(ctx, workItem.TaskID, note)
		}
		return nil
	case swarmies.OutcomeRetry:
		return d.beads.Comment(ctx, workItem.TaskID, ErrorMessage(result))
	default:
		return nil
	}
}

func (d *Dispatcher) normalize(task swarmies.BeadsTask) swarmies.WorkItem {
	item := swarmies.WorkItem{
		TaskID:       task.ID,
		Title:        task.Title,
		Body:         task.Description,
		Labels:       append([]string(nil), task.Labels...),
		Source:       "beads",
		DiscoveredAt: d.now().UTC(),
	}

	if task.RawMetadata == nil {
		return item
	}

	item.ProfileHint = task.RawMetadata["profile"]
	item.Priority = task.RawMetadata["priority"]
	item.RoutingKey = task.RawMetadata["routing_key"]

	return item
}
