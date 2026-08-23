package hoststate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/config"
)

func TestCatalogRetriesFailedMaterialSaveImmediately(t *testing.T) {
	base := t.TempDir()
	stateDir := filepath.Join(base, "state")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(stateDir, "catalog.json")
	catalog, err := openCatalog(path, func(string) (config.Convention, error) {
		return config.Convention{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(stateDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateDir, []byte("blocks catalog directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	observation := heartbeatObservation(123)
	if _, err := catalog.Observe([]Observation{observation}, startedAt); err == nil {
		t.Fatal("expected initial material save to fail")
	}
	if !catalog.materialDirty {
		t.Fatal("failed lifecycle save must remain material-dirty for immediate retry")
	}

	if err := os.Remove(stateDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}

	retryAt := startedAt.Add(2 * time.Second)
	if _, err := catalog.Observe([]Observation{observation}, retryAt); err != nil {
		t.Fatal(err)
	}
	if catalog.materialDirty || catalog.heartbeatDirty {
		t.Fatalf("successful retry left dirty state: material=%v heartbeat=%v", catalog.materialDirty, catalog.heartbeatDirty)
	}

	reloaded, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	repositories := reloaded.Repositories()
	if len(repositories) != 1 || len(repositories[0].Sessions) != 1 || !repositories[0].Sessions[0].Active {
		t.Fatalf("retried lifecycle state was not durable: %#v", repositories)
	}
	if !repositories[0].LastSeenAt.Equal(retryAt) {
		t.Fatalf("retry did not persist latest in-memory heartbeat: %#v", repositories[0])
	}
}
