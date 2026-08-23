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

type fixedSourceControl struct {
	context sourcecontrol.RepositoryContext
}

func (s fixedSourceControl) Inspect(context.Context, string) (sourcecontrol.RepositoryContext, error) {
	return s.context, nil
}

func TestWorkItemDetailProjectsRevisionScopedAcceptance(t *testing.T) {
	repoRoot := t.TempDir()
	writeAcceptanceFixtureProject(t, repoRoot)

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe([]hoststate.Observation{{
		Agent:          "Codex",
		PID:            42,
		RepositoryRoot: repoRoot,
	}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	repo := catalog.Repositories()[0]

	cleanGit := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{
		Path: repoRoot,
		Head: "abc123",
	}}}
	server := NewHostServerWithSources(
		catalog,
		NewHub(),
		"127.0.0.1",
		7331,
		nil,
		fixedSourceControl{context: sourcecontrol.RepositoryContext{Git: cleanGit}},
	)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/project/spec?id="+repo.ID+"&path=H17.md", nil)
	server.projectSpec(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("detail status = %d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`data-specview="acceptance"`,
		`data-acceptance-state="accepted"`,
		`data-check="unit-tests"`,
		`data-check-state="passed"`,
		`data-check="lint"`,
		`git:abc123`,
		`2 evidence records`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("accepted detail missing %q: %s", want, body)
		}
	}
}

func TestWorkItemDetailDoesNotReuseHeadEvidenceForDirtyWorkspace(t *testing.T) {
	repoRoot := t.TempDir()
	writeAcceptanceFixtureProject(t, repoRoot)

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe([]hoststate.Observation{{
		Agent:          "Codex",
		PID:            43,
		RepositoryRoot: repoRoot,
	}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	repo := catalog.Repositories()[0]

	dirtyGit := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{
		Path:       repoRoot,
		Head:       "abc123",
		DirtyCount: 1,
	}}}
	server := NewHostServerWithSources(
		catalog,
		NewHub(),
		"127.0.0.1",
		7331,
		nil,
		fixedSourceControl{context: sourcecontrol.RepositoryContext{Git: dirtyGit}},
	)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/project/spec?id="+repo.ID+"&path=H17.md", nil)
	server.projectSpec(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("detail status = %d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		`data-acceptance-state="waiting"`,
		`unavailable: dirty worktree`,
		`current workspace revision cannot be trusted yet`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("dirty detail missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, `data-acceptance-state="accepted"`) {
		t.Fatalf("dirty workspace inherited accepted state: %s", body)
	}
}

func writeAcceptanceFixtureProject(t *testing.T, repoRoot string) {
	t.Helper()
	config := `version: 1
project:
  name: "Acceptance Web Demo"
  root: "."
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
    - lint
server:
  host: 127.0.0.1
  port: 7331
`
	if err := os.WriteFile(filepath.Join(repoRoot, ".specview.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := `---
specview:
  status: in_progress
---
# H17 Acceptance Policy
`
	if err := os.WriteFile(filepath.Join(repoRoot, "specs", "H17.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}

	evidenceRoot := filepath.Join(repoRoot, ".specview", "evidence")
	if err := os.MkdirAll(evidenceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for index, check := range []string{"unit-tests", "lint"} {
		observed := fmt.Sprintf("2026-08-23T12:00:0%dZ", index)
		record := fmt.Sprintf(`{
  "version": 1,
  "id": "H17-%s",
  "work_item_id": "H17",
  "revision": "git:abc123",
  "check": %q,
  "kind": "test",
  "provider": "fixture",
  "result": "passed",
  "finished_at": %q,
  "observed_at": %q,
  "summary": "fixture passed"
}
`, check, check, observed, observed)
		if err := os.WriteFile(filepath.Join(evidenceRoot, check+".json"), []byte(record), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
