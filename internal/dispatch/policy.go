package dispatch

import (
	a2acore "github.com/a2aproject/a2a-go/a2a"
	"github.com/nikstern/swarmies"
)

type DefaultResultPolicy struct{}

func NewDefaultResultPolicy() DefaultResultPolicy {
	return DefaultResultPolicy{}
}

func (DefaultResultPolicy) Decide(_ swarmies.WorkItem, result a2acore.SendMessageResult) swarmies.OutcomeDecision {
	if structured, ok := ExecutionResult(result); ok {
		switch structured.Outcome {
		case swarmies.OutcomeSuccess:
			return swarmies.OutcomeClose
		case swarmies.OutcomeBlocked, swarmies.OutcomeNeedsInput, swarmies.OutcomeHandoff:
			return swarmies.OutcomeKeep
		case swarmies.OutcomeFailed:
			return swarmies.OutcomeRetry
		}
	}

	switch typed := result.(type) {
	case *a2acore.Message:
		return swarmies.OutcomeClose
	case *a2acore.Task:
		switch typed.Status.State {
		case a2acore.TaskStateCompleted:
			return swarmies.OutcomeClose
		case a2acore.TaskStateFailed, a2acore.TaskStateCanceled, a2acore.TaskStateRejected:
			return swarmies.OutcomeRetry
		default:
			return swarmies.OutcomeKeep
		}
	}

	return swarmies.OutcomeKeep
}
