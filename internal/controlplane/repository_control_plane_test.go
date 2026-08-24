package controlplane

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func TestGetRepositoryControlPlaneComposesExistingProjections(t *testing.T) {
	root := t.TempDir()
	writeControlPlaneProject(t, root)
	initControlPlaneGit(t, root)
	gitContext, err := sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	revisionID := "git:" + gitContext.Worktrees[0].Head
	writeControlPlaneEvidence(t, root, revisionID)

	statePath := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := hoststate.OpenCatalog(statePath)
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
	repository := catalog.Repositories()[0]

	reader := NewReader(
		statePath,
		stubExecutionSource{},
		stubSourceControl{contexts: map[string]sourcecontrol.RepositoryContext{
			filepath.Clean(root): {Git: gitContext},
		}},
	)
	result, err := reader.GetRepositoryControlPlane(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != SchemaVersion || result.RepositoryID != repository.ID {
		t.Fatalf("unexpected control-plane metadata: %#v", result)
	}
	if result.Intent.Total != 1 || result.Intent.InProgress != 1 || result.Intent.Error != "" {
		t.Fatalf("unexpected Intent summary: %#v", result.Intent)
	}
	if result.Execution.Active != 1 || result.Execution.Latest == nil || result.Execution.Latest.SessionID != "live-session" {
		t.Fatalf("unexpected logical Execution summary: %#v", result.Execution)
	}
	if result.Evidence.Total != 1 || result.Evidence.Passed != 1 || result.Evidence.Latest == nil {
		t.Fatalf("unexpected Evidence summary: %#v", result.Evidence)
	}
	if result.Evidence.Latest.Record.Provider != "go-test" || result.Evidence.Latest.WorkItemTitle != "H18 MCP" {
		t.Fatalf("unexpected latest Evidence: %#v", result.Evidence.Latest)
	}
	if !result.Acceptance.Configured || result.Acceptance.Accepted != 1 || result.Acceptance.Waiting != 0 || result.Acceptance.Error != "" {
		t.Fatalf("unexpected Acceptance summary: %#v", result.Acceptance)
	}
	if !result.Acceptance.Revision.Available || result.Acceptance.Revision.Revision != revisionID {
		t.Fatalf("unexpected Acceptance revision: %#v", result.Acceptance.Revision)
	}
}

func TestGetRepositoryControlPlaneKeepsEvidenceWhenIntentUnavailable(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".specview", "evidence"), 0o755); err != nil {
		t.Fatal(err)
	}
	record := `{
  "version": 1,
  "id": "orphan-evidence",
  "work_item_id": "H18",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "fixture",
  "result": "passed",
  "finished_at": "2026-08-24T12:00:00Z",
  "observed_at": "2026-08-24T12:00:00Z"
}
`
	if err := os.WriteFile(filepath.Join(root, ".specview", "evidence", "record.json"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 101, RepositoryRoot: root}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	repository := catalog.Repositories()[0]
	reader := NewReader(statePath, stubExecutionSource{}, stubSourceControl{})

	result, err := reader.GetRepositoryControlPlane(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Intent.Error == "" {
		t.Fatal("Intent must report unavailable specification projection")
	}
	if result.Evidence.Total != 1 || result.Evidence.Passed != 1 || result.Evidence.Latest == nil {
		t.Fatalf("native Evidence must survive Intent failure: %#v", result.Evidence)
	}
	if result.Evidence.WorkItemError == "" {
		t.Fatalf("Evidence must expose degraded WorkItem enrichment: %#v", result.Evidence)
	}
	if result.Acceptance.Error == "" {
		t.Fatalf("Acceptance must degrade independently when WorkItems are unavailable: %#v", result.Acceptance)
	}
}
