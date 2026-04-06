package dispatch

import "github.com/nikstern/swarmies"

type DefaultResultPolicy struct{}

func NewDefaultResultPolicy() DefaultResultPolicy {
	return DefaultResultPolicy{}
}

func (DefaultResultPolicy) Decide(_ swarmies.WorkItem, result swarmies.DispatchResult) swarmies.OutcomeDecision {
	switch result.State {
	case swarmies.StateSucceeded:
		return swarmies.OutcomeClose
	case swarmies.StateFailed:
		return swarmies.OutcomeRetry
	default:
		return swarmies.OutcomeKeep
	}
}
