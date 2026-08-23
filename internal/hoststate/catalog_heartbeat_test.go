package hoststate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/config"
)

// TestCatalogHeartbeatPersistenceBaseline characterizes the v0.0.1 catalog
// behavior before any post-release write-coalescing migration. A repeated live
// observation advances LastSeenAt and currently persists a new catalog snapshot.
func TestCatalogHeartbeatPersistenceBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	detector := func(string) (config.Convention, error) {
		return config.Convention{
			Adapter:    config.AdapterSpecview,
			Label:      "Specview",
			Path:       "specs",
			Recognized: true,
			Supported:  true,
		}, nil
	}
	catalog, err := openCatalog(path, detector)
	if err != nil {
		t.Fatal(err)
	}

	observation := Observation{
		Agent:          "Codex",
		PID:            123,
		RepositoryRoot: "/work/specview",
	}
	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	if _, err := catalog.Observe([]Observation{observation}, startedAt); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	heartbeatAt := startedAt.Add(2 * time.Second)
	if _, err := catalog.Observe([]Observation{observation}, heartbeatAt); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("expected the v0.0.1 baseline to persist a heartbeat-only LastSeenAt change")
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 || !repositories[0].LastSeenAt.Equal(heartbeatAt) {
		t.Fatalf("heartbeat LastSeenAt was not retained in memory: %#v", repositories)
	}
}
