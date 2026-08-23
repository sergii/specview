package web

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

type emptyExecutionSource struct{}

func (emptyExecutionSource) Sessions() ([]hoststate.ExecutionSession, error) {
	return nil, nil
}

func TestHostMaterialFingerprintIgnoresHeartbeatLastSeen(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "wms")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	observation := hoststate.Observation{Agent: "Codex", PID: 101, RepositoryRoot: root}
	now := time.Date(2026, 8, 22, 2, 0, 0, 0, time.Local)
	if _, err := catalog.Observe([]hoststate.Observation{observation}, now); err != nil {
		t.Fatal(err)
	}

	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, emptyExecutionSource{}, nil)
	before, err := server.hostMaterialFingerprint()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := catalog.Observe([]hoststate.Observation{observation}, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	after, err := server.hostMaterialFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("heartbeat-only last_seen updates must not change the material host fingerprint")
	}
}

func TestHostMaterialFingerprintChangesWhenExecutionStops(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "wms")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 22, 2, 0, 0, 0, time.Local)
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 101, RepositoryRoot: root}}, now); err != nil {
		t.Fatal(err)
	}

	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, emptyExecutionSource{}, nil)
	before, err := server.hostMaterialFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe(nil, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	after, err := server.hostMaterialFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("execution stop must change the material host fingerprint")
	}
}

func TestProjectMaterialFingerprintChangesWithGitState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wms")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", root, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 22, 2, 0, 0, 0, time.Local)
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 101, RepositoryRoot: root}}, now); err != nil {
		t.Fatal(err)
	}
	repository := catalog.Repositories()[0]

	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, emptyExecutionSource{}, nil)
	before, err := server.projectMaterialFingerprint(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# WMS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := server.projectMaterialFingerprint(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("Git dirty-state change must change the material project fingerprint")
	}
}

func TestProjectMaterialFingerprintChangesWithEvidence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "acceptance-demo")
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", root, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	config := `version: 1
project:
  name: "Acceptance Demo"
  root: "."
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
	if err := os.WriteFile(filepath.Join(root, "specs", "H17.md"), []byte("# H17\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	evidenceRoot := filepath.Join(root, ".specview", "evidence")
	if err := os.MkdirAll(evidenceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(evidenceRoot, "tests.json")
	beforeEvidence := `{
  "version": 1,
  "id": "H17-tests",
  "work_item_id": "H17",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "fixture",
  "result": "passed",
  "finished_at": "2026-08-23T12:00:00Z",
  "observed_at": "2026-08-23T12:00:00Z",
  "summary": "passed once"
}
`
	if err := os.WriteFile(evidencePath, []byte(beforeEvidence), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 101, RepositoryRoot: root}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	repository := catalog.Repositories()[0]
	server := NewHostServerWithSources(catalog, NewHub(), "127.0.0.1", 7331, emptyExecutionSource{}, nil)

	before, err := server.projectMaterialFingerprint(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	afterEvidence := []byte(string(beforeEvidence[:len(beforeEvidence)-2]) + ",\n  \"summary\": \"passed twice\"\n}\n")
	// Rewrite the record with a different observed fact while keeping the same identity.
	afterEvidence = []byte(`{
  "version": 1,
  "id": "H17-tests",
  "work_item_id": "H17",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "fixture",
  "result": "passed",
  "finished_at": "2026-08-23T12:00:00Z",
  "observed_at": "2026-08-23T12:00:00Z",
  "summary": "passed twice"
}
`)
	if err := os.WriteFile(evidencePath, afterEvidence, 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := server.projectMaterialFingerprint(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("Evidence change must change the material project fingerprint")
	}
}
