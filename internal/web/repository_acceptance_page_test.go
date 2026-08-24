package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

type acceptanceOverviewSourceControl struct {
	root string
}

func (s acceptanceOverviewSourceControl) Inspect(context.Context, string) (sourcecontrol.RepositoryContext, error) {
	return sourcecontrol.RepositoryContext{Git: sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: s.root, Head: "abc123"}}}}, nil
}

func TestRepositoryAcceptancePageRendersAcceptedOverview(t *testing.T) {
	root := t.TempDir()
	if err := writeAcceptanceOverviewFixture(root); err != nil {
		t.Fatal(err)
	}
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 14, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{ID: "acceptance-session", Adapter: "codex", Agent: "Codex", RepositoryRoot: root, WorktreeRoot: root, CWD: root, ProcessIDs: []int{71}, StartedAt: now}}, now); err != nil {
		t.Fatal(err)
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repositories))
	}

	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, nil, acceptanceOverviewSourceControl{root: root})
	recorder := httptest.NewRecorder()
	server.repositoryAcceptancePage(recorder, httptest.NewRequest(http.MethodGet, "/project/acceptance?id="+repositories[0].ID, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-specview="acceptance-overview"`,
		`data-acceptance-configured="true"`,
		`data-accepted="1"`,
		`data-evidence-count="1"`,
		`data-work-item="H33"`,
		`data-acceptance-state="accepted"`,
		`git:abc123`,
		`href="/project/spec?id=` + repositories[0].ID,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("acceptance overview missing %q; body=%s", want, body)
		}
	}
}

func TestRepositoryAcceptancePageRejectsUnknownRepository(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.repositoryAcceptancePage(recorder, httptest.NewRequest(http.MethodGet, "/project/acceptance?id=missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func writeAcceptanceOverviewFixture(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".specview", "evidence"), 0o755); err != nil {
		return err
	}
	config := `version: 2
project:
  name: "Acceptance Fixture"
  root: "."
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
`
	if err := os.WriteFile(filepath.Join(root, ".specview.yaml"), []byte(config), 0o644); err != nil {
		return err
	}
	spec := `---
specview:
  status: in_progress
---

# H33 Repository Acceptance
`
	if err := os.WriteFile(filepath.Join(root, "specs", "H33.md"), []byte(spec), 0o644); err != nil {
		return err
	}
	observedAt := "2026-08-24T14:00:00Z"
	record := fmt.Sprintf(`{
  "version": 1,
  "id": "H33-unit-tests",
  "work_item_id": "H33",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "fixture",
  "result": "passed",
  "finished_at": %q,
  "observed_at": %q,
  "summary": "tests passed"
}
`, observedAt, observedAt)
	return os.WriteFile(filepath.Join(root, ".specview", "evidence", "unit-tests.json"), []byte(record), 0o644)
}
