package acceptance

import (
	"testing"
	"time"

	"github.com/sergii/specview/internal/evidence"
)

func TestEvaluateUnconfiguredPolicy(t *testing.T) {
	decision, err := Evaluate(Policy{}, "", "", nil)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.State != StateUnconfigured {
		t.Fatalf("State = %q, want %q", decision.State, StateUnconfigured)
	}
}

func TestEvaluateAcceptedWhenAllRequiredChecksPass(t *testing.T) {
	policy := Policy{Required: []Requirement{{Check: "unit-tests"}, {Check: "lint"}}}
	records := []evidence.Record{
		record("tests", "ATS-003", "git:abc", "unit-tests", evidence.ResultPassed, "rspec", 1),
		record("lint", "ATS-003", "git:abc", "lint", evidence.ResultPassed, "rubocop", 2),
	}

	decision, err := Evaluate(policy, "ATS-003", "git:abc", records)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.State != StateAccepted {
		t.Fatalf("State = %q, want %q", decision.State, StateAccepted)
	}
	if len(decision.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(decision.Checks))
	}
}

func TestEvaluateWaitingForMissingStaleOrRunningEvidence(t *testing.T) {
	tests := []struct {
		name      string
		records   []evidence.Record
		wantState CheckState
	}{
		{name: "missing", wantState: CheckMissing},
		{
			name:      "stale",
			records:   []evidence.Record{record("old", "ATS-003", "git:old", "unit-tests", evidence.ResultPassed, "rspec", 1)},
			wantState: CheckStale,
		},
		{
			name:      "queued",
			records:   []evidence.Record{record("queued", "ATS-003", "git:abc", "unit-tests", evidence.ResultQueued, "rspec", 1)},
			wantState: CheckQueued,
		},
		{
			name:      "running",
			records:   []evidence.Record{record("running", "ATS-003", "git:abc", "unit-tests", evidence.ResultRunning, "rspec", 1)},
			wantState: CheckRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := Evaluate(Policy{Required: []Requirement{{Check: "unit-tests"}}}, "ATS-003", "git:abc", tt.records)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if decision.State != StateWaiting {
				t.Fatalf("State = %q, want %q", decision.State, StateWaiting)
			}
			if decision.Checks[0].State != tt.wantState {
				t.Fatalf("check state = %q, want %q", decision.Checks[0].State, tt.wantState)
			}
		})
	}
}

func TestEvaluateBlockedByFailedErrorInvalidOrDisallowedSkipped(t *testing.T) {
	tests := []struct {
		name      string
		record    evidence.Record
		wantState CheckState
	}{
		{name: "failed", record: record("failed", "ATS-003", "git:abc", "unit-tests", evidence.ResultFailed, "rspec", 1), wantState: CheckFailed},
		{name: "error", record: record("error", "ATS-003", "git:abc", "unit-tests", evidence.ResultError, "rspec", 1), wantState: CheckError},
		{name: "skipped", record: record("skipped", "ATS-003", "git:abc", "unit-tests", evidence.ResultSkipped, "rspec", 1), wantState: CheckSkipped},
		{name: "invalid", record: invalidRecord("invalid", "ATS-003", "git:abc", "unit-tests", 1), wantState: CheckInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := Evaluate(Policy{Required: []Requirement{{Check: "unit-tests"}}}, "ATS-003", "git:abc", []evidence.Record{tt.record})
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if decision.State != StateBlocked {
				t.Fatalf("State = %q, want %q", decision.State, StateBlocked)
			}
			if decision.Checks[0].State != tt.wantState {
				t.Fatalf("check state = %q, want %q", decision.Checks[0].State, tt.wantState)
			}
		})
	}
}

func TestEvaluateAllowsExplicitlySkippedRequirement(t *testing.T) {
	policy := Policy{Required: []Requirement{{Check: "hardware-in-loop", AllowSkipped: true}}}
	records := []evidence.Record{record("skip", "IOT-001", "git:abc", "hardware-in-loop", evidence.ResultSkipped, "rig", 1)}

	decision, err := Evaluate(policy, "IOT-001", "git:abc", records)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.State != StateAccepted {
		t.Fatalf("State = %q, want %q", decision.State, StateAccepted)
	}
	if decision.Checks[0].State != CheckSkipped {
		t.Fatalf("check state = %q, want %q", decision.Checks[0].State, CheckSkipped)
	}
}

func TestEvaluateUsesLatestEvidenceForCurrentRevision(t *testing.T) {
	records := []evidence.Record{
		record("pass", "ATS-003", "git:abc", "unit-tests", evidence.ResultPassed, "rspec", 1),
		record("fail", "ATS-003", "git:abc", "unit-tests", evidence.ResultFailed, "rspec", 2),
		record("other-work", "ATS-004", "git:abc", "unit-tests", evidence.ResultPassed, "rspec", 3),
	}

	decision, err := Evaluate(Policy{Required: []Requirement{{Check: "unit-tests"}}}, "ATS-003", "git:abc", records)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.State != StateBlocked {
		t.Fatalf("State = %q, want %q", decision.State, StateBlocked)
	}
	if decision.Checks[0].EvidenceID != "fail" {
		t.Fatalf("EvidenceID = %q, want %q", decision.Checks[0].EvidenceID, "fail")
	}
}

func TestEvaluateBlockedTakesPrecedenceOverWaiting(t *testing.T) {
	policy := Policy{Required: []Requirement{{Check: "unit-tests"}, {Check: "lint"}}}
	records := []evidence.Record{record("failed", "ATS-003", "git:abc", "unit-tests", evidence.ResultFailed, "rspec", 1)}

	decision, err := Evaluate(policy, "ATS-003", "git:abc", records)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.State != StateBlocked {
		t.Fatalf("State = %q, want %q", decision.State, StateBlocked)
	}
}

func TestPolicyValidateRejectsDuplicateOrBlankChecks(t *testing.T) {
	tests := []Policy{
		{Required: []Requirement{{Check: ""}}},
		{Required: []Requirement{{Check: "unit-tests"}, {Check: "unit-tests"}}},
	}

	for _, policy := range tests {
		if err := policy.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	}
}

func TestEvaluateRequiresIdentityWhenPolicyConfigured(t *testing.T) {
	policy := Policy{Required: []Requirement{{Check: "unit-tests"}}}

	if _, err := Evaluate(policy, "", "git:abc", nil); err == nil {
		t.Fatal("Evaluate() missing work item error = nil, want error")
	}
	if _, err := Evaluate(policy, "ATS-003", "", nil); err == nil {
		t.Fatal("Evaluate() missing revision error = nil, want error")
	}
}

func record(id, workItemID, revision, check string, result evidence.Result, provider string, second int) evidence.Record {
	observed := time.Date(2026, 8, 23, 12, 0, second, 0, time.UTC)
	record := evidence.Record{
		Version:    1,
		ID:         id,
		WorkItemID: workItemID,
		Revision:   revision,
		Check:      check,
		Kind:       evidence.KindTest,
		Provider:   provider,
		Result:     result,
		ObservedAt: observed,
	}
	if result == evidence.ResultPassed || result == evidence.ResultFailed || result == evidence.ResultError || result == evidence.ResultSkipped {
		finished := observed
		record.FinishedAt = &finished
	}
	return record
}

func invalidRecord(id, workItemID, revision, check string, second int) evidence.Record {
	record := record(id, workItemID, revision, check, evidence.ResultPassed, "rspec", second)
	record.Error = "invalid evidence"
	return record
}
