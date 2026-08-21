package web

import (
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
	before, err := server.projectMaterialFingerprint(t.Context(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# WMS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := server.projectMaterialFingerprint(t.Context(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("Git dirty-state change must change the material project fingerprint")
	}
}
