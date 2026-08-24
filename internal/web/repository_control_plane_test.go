package web

import (
	"context"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

type controlPlaneSourceControl struct {
	root string
}

func (s controlPlaneSourceControl) Inspect(context.Context, string) (sourcecontrol.RepositoryContext, error) {
	return sourcecontrol.RepositoryContext{Git: sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: s.root, Head: "abc123"}}}}, nil
}

func TestRepositoryControlPlaneSummaryComposesExistingAuthorities(t *testing.T) {
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

	repositories := catalog.Repositories()
	if len(repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repositories))
	}
	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, nil, controlPlaneSourceControl{root: root})
	data, err := server.loadProject(context.Background(), repositories[0])
	if err != nil {
		t.Fatal(err)
	}
	summary := data.ControlPlaneSummary()

	if summary.Intent.Total != 1 || summary.Intent.InProgress != 1 || summary.Intent.Invalid != 0 {
		t.Fatalf("intent summary = %#v", summary.Intent)
	}
	if summary.Execution.Active != 1 || !summary.Execution.HasLatest || summary.Execution.Latest.SessionID != "live-session" || !summary.Execution.Latest.Active {
		t.Fatalf("execution summary = %#v", summary.Execution)
	}
	if summary.Evidence.Error != "" || summary.Evidence.Overview.Total != 1 || summary.Evidence.Overview.Passed != 1 || !summary.Evidence.HasLatest || summary.Evidence.Latest.Record.Check != "unit-tests" {
		t.Fatalf("evidence summary = %#v", summary.Evidence)
	}
	if summary.Acceptance.Error != "" || !summary.Acceptance.Overview.Configured || summary.Acceptance.Overview.Accepted != 1 || summary.Acceptance.Overview.Blocked != 0 {
		t.Fatalf("acceptance summary = %#v", summary.Acceptance)
	}
}

func TestRepositoryControlPlaneSummaryKeepsEvidenceVisibleWhenIntentUnavailable(t *testing.T) {
	root := t.TempDir()
	if err := writeAcceptanceOverviewFixture(root); err != nil {
		t.Fatal(err)
	}
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 14, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "evidence-only-session",
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

	repositories := catalog.Repositories()
	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, nil, controlPlaneSourceControl{root: root})
	data, err := server.loadProject(context.Background(), repositories[0])
	if err != nil {
		t.Fatal(err)
	}

	data.Convention.Recognized = false
	data.New = nil
	data.InProgress = nil
	data.Done = nil
	data.Invalid = nil
	data.Total = 0
	summary := data.ControlPlaneSummary()
	if summary.Evidence.Overview.Total != 1 || summary.Evidence.Overview.Passed != 1 {
		t.Fatalf("evidence disappeared with unavailable intent: %#v", summary.Evidence)
	}
}
