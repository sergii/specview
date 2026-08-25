package web

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

type hostControlPlaneSearcher struct {
	repositoryID string
}

func (s hostControlPlaneSearcher) SearchRepositoryIDs(context.Context, string, int) ([]string, error) {
	return []string{s.repositoryID}, nil
}

func TestHostControlPlaneComposesRepositoryAuthorities(t *testing.T) {
	root := t.TempDir()
	if err := writeAcceptanceOverviewFixture(root); err != nil {
		t.Fatal(err)
	}
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	endedAt := time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "ended-session",
		Adapter:        "claude-code",
		Agent:          "Claude",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{31},
		StartedAt:      endedAt.Add(-10 * time.Minute),
	}}, endedAt); err != nil {
		t.Fatal(err)
	}
	liveAt := endedAt.Add(time.Hour)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "live-session",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{41},
		StartedAt:      liveAt.Add(-5 * time.Minute),
	}}, liveAt); err != nil {
		t.Fatal(err)
	}

	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, nil, controlPlaneSourceControl{root: root})
	summary := server.hostControlPlane(context.Background(), catalog.Repositories())

	if summary.Intent.ManagedRepositories != 1 || summary.Intent.WorkItems != 1 || summary.Intent.InProgress != 1 || summary.Intent.Invalid != 0 {
		t.Fatalf("intent summary = %#v", summary.Intent)
	}
	if summary.Execution.ActiveSessions != 1 || summary.Execution.ActiveRepositories != 1 || !summary.Execution.HasLatest || summary.Execution.Latest.SessionID != "live-session" {
		t.Fatalf("execution summary = %#v", summary.Execution)
	}
	if summary.Evidence.Total != 1 || summary.Evidence.Passed != 1 || summary.Evidence.Failed != 0 || !summary.Evidence.HasLatest {
		t.Fatalf("evidence summary = %#v", summary.Evidence)
	}
	if summary.Evidence.Latest.RepositoryID == "" || summary.Evidence.Latest.Entry.Record.Check != "unit-tests" {
		t.Fatalf("latest evidence = %#v", summary.Evidence.Latest)
	}
	if summary.Acceptance.ConfiguredRepositories != 1 || summary.Acceptance.Accepted != 1 || summary.Acceptance.Blocked != 0 || summary.Acceptance.Waiting != 0 {
		t.Fatalf("acceptance summary = %#v", summary.Acceptance)
	}
	if len(summary.Attention) != 0 {
		t.Fatalf("attention = %#v, want none", summary.Attention)
	}
}

func TestHostControlPlaneRemainsGlobalWhenRepositorySearchIsActive(t *testing.T) {
	root := t.TempDir()
	if err := writeAcceptanceOverviewFixture(root); err != nil {
		t.Fatal(err)
	}
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 13, 30, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "search-session",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{45},
		StartedAt:      now,
	}}, now); err != nil {
		t.Fatal(err)
	}
	repository := catalog.Repositories()[0]
	server := NewHostServerWithSources(
		catalog,
		NewHub(),
		"127.0.0.1",
		7331,
		nil,
		controlPlaneSourceControl{root: root},
		hostControlPlaneSearcher{repositoryID: repository.ID},
	)

	data := server.loadHostData(context.Background(), "specview", now)
	if !data.Filtered || len(data.Results) != 1 {
		t.Fatalf("search projection = %#v", data)
	}
	if data.ControlPlane.Intent.WorkItems != 1 || data.ControlPlane.Execution.ActiveSessions != 1 || data.ControlPlane.Evidence.Total != 1 {
		t.Fatalf("search changed global control plane: %#v", data.ControlPlane)
	}
}

func TestHostControlPlaneSurfacesFailedEvidenceAndBlockedAcceptance(t *testing.T) {
	root := t.TempDir()
	if err := writeAcceptanceOverviewFixture(root); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(root, ".specview", "evidence", "unit-tests.json")
	body, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	failed := strings.Replace(string(body), `"result": "passed"`, `"result": "failed"`, 1)
	if err := os.WriteFile(evidencePath, []byte(failed), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 14, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "failed-session",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{51},
		StartedAt:      now,
	}}, now); err != nil {
		t.Fatal(err)
	}

	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, nil, controlPlaneSourceControl{root: root})
	summary := server.hostControlPlane(context.Background(), catalog.Repositories())
	if summary.Evidence.Failed != 1 || summary.Evidence.AffectedRepositories != 1 {
		t.Fatalf("evidence summary = %#v", summary.Evidence)
	}
	if summary.Acceptance.Blocked != 1 || summary.Acceptance.Waiting != 0 {
		t.Fatalf("acceptance summary = %#v", summary.Acceptance)
	}
	if len(summary.Attention) != 1 {
		t.Fatalf("attention = %#v, want one repository", summary.Attention)
	}
	signals := strings.Join(summary.Attention[0].Signals, " | ")
	if !strings.Contains(signals, "1 failed Evidence record") || !strings.Contains(signals, "1 blocked Acceptance item") {
		t.Fatalf("attention signals = %q", signals)
	}
}

func TestHostControlPlaneDoesNotTreatUnrecognizedRepositoryAsAcceptanceFailure(t *testing.T) {
	root := t.TempDir()
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 15, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "plain-repository-session",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{61},
		StartedAt:      now,
	}}, now); err != nil {
		t.Fatal(err)
	}

	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	summary := server.hostControlPlane(context.Background(), catalog.Repositories())
	if summary.Execution.ActiveSessions != 1 || summary.Intent.ManagedRepositories != 0 {
		t.Fatalf("unexpected host summary: %#v", summary)
	}
	if summary.Acceptance.UnavailableRepositories != 0 || len(summary.Attention) != 0 {
		t.Fatalf("plain repository became false attention: %#v", summary)
	}
}
