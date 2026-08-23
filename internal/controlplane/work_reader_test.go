package controlplane

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func TestWorkEvidenceAcceptanceReadContracts(t *testing.T) {
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
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 101, RepositoryRoot: root}}, time.Now()); err != nil {
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

	workItem, err := reader.GetWorkItem(context.Background(), repository.ID, "H18")
	if err != nil {
		t.Fatal(err)
	}
	if workItem.WorkItem.Title != "H18 MCP" || workItem.WorkItem.Status != "in_progress" {
		t.Fatalf("unexpected work item: %#v", workItem.WorkItem)
	}
	if workItem.WorkItem.Body == "" {
		t.Fatal("work item body must be available to agent clients")
	}

	evidence, err := reader.GetEvidence(context.Background(), repository.ID, "H18")
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Records) != 1 || evidence.Records[0].Revision != revisionID || evidence.Records[0].Provider != "go-test" {
		t.Fatalf("unexpected evidence: %#v", evidence.Records)
	}

	acceptance, err := reader.GetAcceptance(context.Background(), repository.ID, "H18")
	if err != nil {
		t.Fatal(err)
	}
	if acceptance.Decision.State != "accepted" || !acceptance.Revision.Available || acceptance.Revision.Revision != revisionID {
		t.Fatalf("unexpected acceptance: %#v", acceptance)
	}
	if len(acceptance.Policy.Required) != 1 || acceptance.Policy.Required[0].Check != "unit-tests" {
		t.Fatalf("unexpected acceptance policy: %#v", acceptance.Policy)
	}
	if len(acceptance.Decision.Checks) != 1 || acceptance.Decision.Checks[0].EvidenceID != "H18-tests" {
		t.Fatalf("unexpected acceptance checks: %#v", acceptance.Decision.Checks)
	}
}

func TestGetAcceptanceDoesNotRequireGitWhenPolicyUnconfigured(t *testing.T) {
	root := t.TempDir()
	config := `version: 1
project:
  name: No Policy
  root: .
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
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
	if err := os.WriteFile(filepath.Join(root, "specs", "H18.md"), []byte("# H18 MCP\n"), 0o644); err != nil {
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
	reader := NewReader(statePath, stubExecutionSource{}, stubSourceControl{err: context.DeadlineExceeded})

	result, err := reader.GetAcceptance(context.Background(), repository.ID, "H18")
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision.State != "unconfigured" {
		t.Fatalf("unconfigured policy must not require Git: %#v", result)
	}
}

func writeControlPlaneProject(t *testing.T, root string) {
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

Expose normalized facts to agents.
`
	if err := os.WriteFile(filepath.Join(root, "specs", "H18.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".specview/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func initControlPlaneGit(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "specview@example.test"},
		{"config", "user.name", "Specview Test"},
		{"add", "."},
		{"commit", "-m", "fixture"},
	} {
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
}

func writeControlPlaneEvidence(t *testing.T, root, revisionID string) {
	t.Helper()
	dir := filepath.Join(root, ".specview", "evidence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	observedAt := "2026-08-23T18:00:00Z"
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
`, revisionID, observedAt, observedAt)
	if err := os.WriteFile(filepath.Join(dir, "tests.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
