package projectstate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/revision"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func TestProjectStateLoadsWorkItemEvidenceAndAcceptance(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root)
	initGitFixture(t, root)

	project, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	item, err := project.FindWorkItem("H18")
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "H18 MCP" || item.WorkItemID != "H18" {
		t.Fatalf("unexpected work item: %#v", item)
	}

	git, err := sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(git.Worktrees) != 1 {
		t.Fatalf("worktrees = %#v", git.Worktrees)
	}
	revisionID := "git:" + git.Worktrees[0].Head
	writeEvidenceFixture(t, root, revisionID)

	records, err := project.Evidence("H18")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Check != "unit-tests" {
		t.Fatalf("unexpected evidence: %#v", records)
	}

	result, err := project.EvaluateAcceptance("H18", git)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision.State != acceptance.StateAccepted || result.Revision.Revision != revisionID {
		t.Fatalf("unexpected acceptance: %#v", result)
	}
}

func TestProjectStateDirtyWorktreeFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root)
	initGitFixture(t, root)

	project, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	git, err := sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	writeEvidenceFixture(t, root, "git:"+git.Worktrees[0].Head)

	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git, err = sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := project.EvaluateAcceptance("H18", git)
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision.State != acceptance.StateWaiting || !result.EvaluationPending {
		t.Fatalf("dirty worktree must wait: %#v", result)
	}
	if result.Revision.Reason != revision.ReasonDirtyWorktree {
		t.Fatalf("revision reason = %q", result.Revision.Reason)
	}
}

func writeProjectFixture(t *testing.T, root string) {
	t.Helper()
	config := `version: 1
project:
  name: H18 Fixture
  root: .
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
server:
  host: 127.0.0.1
  port: 7331
`
	if err := os.WriteFile(filepath.Join(root, ".specview.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := `---
specview:
  status: in_progress
---

# H18 MCP

Read-only agent interface.
`
	if err := os.WriteFile(filepath.Join(root, "specs", "H18.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("clean\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".specview/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func initGitFixture(t *testing.T, root string) {
	t.Helper()
	commands := [][]string{
		{"init"},
		{"config", "user.email", "specview@example.test"},
		{"config", "user.name", "Specview Test"},
		{"add", "."},
		{"commit", "-m", "fixture"},
	}
	for _, args := range commands {
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
}

func writeEvidenceFixture(t *testing.T, root, revisionID string) {
	t.Helper()
	path := filepath.Join(root, ".specview", "evidence")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	body := fmt.Sprintf(`{
  "version": 1,
  "id": "H18-tests",
  "work_item_id": "H18",
  "revision": %q,
  "check": "unit-tests",
  "kind": "test",
  "provider": "go-test",
  "result": "passed",
  "finished_at": %q,
  "observed_at": %q,
  "summary": "tests passed"
}
`, revisionID, observedAt.Format(time.RFC3339), observedAt.Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(path, "tests.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
