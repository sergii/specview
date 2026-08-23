package acceptance

import (
	"fmt"
	"strings"

	"github.com/sergii/specview/internal/evidence"
)

type State string

const (
	StateUnconfigured State = "unconfigured"
	StateWaiting      State = "waiting"
	StateBlocked      State = "blocked"
	StateAccepted     State = "accepted"
)

type CheckState string

const (
	CheckMissing CheckState = "missing"
	CheckStale   CheckState = "stale"
	CheckQueued  CheckState = "queued"
	CheckRunning CheckState = "running"
	CheckPassed  CheckState = "passed"
	CheckFailed  CheckState = "failed"
	CheckError   CheckState = "error"
	CheckSkipped CheckState = "skipped"
	CheckInvalid CheckState = "invalid"
)

type Requirement struct {
	Check        string
	AllowSkipped bool
}

type Policy struct {
	Required []Requirement
}

func (p Policy) Validate() error {
	seen := make(map[string]struct{}, len(p.Required))
	for _, requirement := range p.Required {
		check := strings.TrimSpace(requirement.Check)
		if check == "" {
			return fmt.Errorf("acceptance requirement check is required")
		}
		if _, exists := seen[check]; exists {
			return fmt.Errorf("duplicate acceptance requirement %q", check)
		}
		seen[check] = struct{}{}
	}
	return nil
}

type CheckDecision struct {
	Check      string
	State      CheckState
	Provider   string
	EvidenceID string
	Summary    string
}

type Decision struct {
	WorkItemID string
	Revision   string
	State      State
	Checks     []CheckDecision
}

func Evaluate(policy Policy, workItemID, revision string, records []evidence.Record) (Decision, error) {
	if err := policy.Validate(); err != nil {
		return Decision{}, err
	}

	decision := Decision{
		WorkItemID: workItemID,
		Revision:   revision,
		State:      StateUnconfigured,
	}
	if len(policy.Required) == 0 {
		return decision, nil
	}
	if strings.TrimSpace(workItemID) == "" {
		return Decision{}, fmt.Errorf("work item id is required for acceptance evaluation")
	}
	if strings.TrimSpace(revision) == "" {
		return Decision{}, fmt.Errorf("revision is required for acceptance evaluation")
	}

	decision.State = StateAccepted
	decision.Checks = make([]CheckDecision, 0, len(policy.Required))
	for _, requirement := range policy.Required {
		check := evaluateRequirement(requirement, workItemID, revision, records)
		decision.Checks = append(decision.Checks, check)

		switch {
		case blocks(requirement, check.State):
			decision.State = StateBlocked
		case decision.State != StateBlocked && !satisfies(requirement, check.State):
			decision.State = StateWaiting
		}
	}

	return decision, nil
}

func evaluateRequirement(requirement Requirement, workItemID, revision string, records []evidence.Record) CheckDecision {
	var current *evidence.Record
	hasStale := false

	for i := range records {
		record := &records[i]
		if record.WorkItemID != workItemID || record.Check != requirement.Check {
			continue
		}
		if record.Revision != revision {
			hasStale = true
			continue
		}
		if current == nil || newer(record, current) {
			current = record
		}
	}

	if current == nil {
		state := CheckMissing
		if hasStale {
			state = CheckStale
		}
		return CheckDecision{Check: requirement.Check, State: state}
	}

	return CheckDecision{
		Check:      requirement.Check,
		State:      stateForRecord(*current),
		Provider:   current.Provider,
		EvidenceID: current.ID,
		Summary:    current.Summary,
	}
}

func newer(candidate, current *evidence.Record) bool {
	if candidate.ObservedAt.After(current.ObservedAt) {
		return true
	}
	if candidate.ObservedAt.Equal(current.ObservedAt) {
		return candidate.ID > current.ID
	}
	return false
}

func stateForRecord(record evidence.Record) CheckState {
	if record.Error != "" {
		return CheckInvalid
	}

	switch record.Result {
	case evidence.ResultQueued:
		return CheckQueued
	case evidence.ResultRunning:
		return CheckRunning
	case evidence.ResultPassed:
		return CheckPassed
	case evidence.ResultFailed:
		return CheckFailed
	case evidence.ResultError:
		return CheckError
	case evidence.ResultSkipped:
		return CheckSkipped
	default:
		return CheckInvalid
	}
}

func satisfies(requirement Requirement, state CheckState) bool {
	return state == CheckPassed || (state == CheckSkipped && requirement.AllowSkipped)
}

func blocks(requirement Requirement, state CheckState) bool {
	switch state {
	case CheckFailed, CheckError, CheckInvalid:
		return true
	case CheckSkipped:
		return !requirement.AllowSkipped
	default:
		return false
	}
}
