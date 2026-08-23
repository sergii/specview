package hoststate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/config"
)

func heartbeatTestCatalog(t *testing.T) (*Catalog, string) {
	t.Helper()
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
	return catalog, path
}

func heartbeatObservation(pid int) Observation {
	return Observation{
		Agent:          "Codex",
		PID:            pid,
		RepositoryRoot: "/work/specview",
	}
}

func readCatalogBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCatalogCoalescesHeartbeatPersistence(t *testing.T) {
	catalog, path := heartbeatTestCatalog(t)
	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	observation := heartbeatObservation(123)

	if changed, err := catalog.Observe([]Observation{observation}, startedAt); err != nil || !changed {
		t.Fatalf("initial observation changed=%v err=%v", changed, err)
	}
	initial := readCatalogBytes(t, path)

	heartbeatAt := startedAt.Add(2 * time.Second)
	if changed, err := catalog.Observe([]Observation{observation}, heartbeatAt); err != nil || !changed {
		t.Fatalf("heartbeat changed=%v err=%v", changed, err)
	}
	if got := readCatalogBytes(t, path); !bytes.Equal(initial, got) {
		t.Fatal("heartbeat inside coalescing window rewrote catalog.json")
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 || !repositories[0].LastSeenAt.Equal(heartbeatAt) || !repositories[0].Sessions[0].LastSeenAt.Equal(heartbeatAt) {
		t.Fatalf("heartbeat was not retained in memory: %#v", repositories)
	}

	boundaryAt := startedAt.Add(heartbeatPersistInterval)
	if changed, err := catalog.Observe([]Observation{observation}, boundaryAt); err != nil || !changed {
		t.Fatalf("boundary heartbeat changed=%v err=%v", changed, err)
	}
	persisted := readCatalogBytes(t, path)
	if bytes.Equal(initial, persisted) {
		t.Fatal("heartbeat at coalescing boundary did not persist catalog.json")
	}

	reloaded, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Repositories()
	if len(got) != 1 || !got[0].LastSeenAt.Equal(boundaryAt) || !got[0].Sessions[0].LastSeenAt.Equal(boundaryAt) {
		t.Fatalf("coalesced heartbeat was not persisted: %#v", got)
	}
}

func TestCatalogMaterialChangeBypassesHeartbeatThrottle(t *testing.T) {
	catalog, path := heartbeatTestCatalog(t)
	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	first := heartbeatObservation(123)

	if _, err := catalog.Observe([]Observation{first}, startedAt); err != nil {
		t.Fatal(err)
	}
	initial := readCatalogBytes(t, path)
	if _, err := catalog.Observe([]Observation{first}, startedAt.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := readCatalogBytes(t, path); !bytes.Equal(initial, got) {
		t.Fatal("precondition failed: heartbeat should still be coalesced")
	}

	materialAt := startedAt.Add(4 * time.Second)
	second := heartbeatObservation(456)
	if _, err := catalog.Observe([]Observation{first, second}, materialAt); err != nil {
		t.Fatal(err)
	}
	if got := readCatalogBytes(t, path); bytes.Equal(initial, got) {
		t.Fatal("new session did not bypass heartbeat throttle")
	}

	reloaded, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	repositories := reloaded.Repositories()
	if len(repositories) != 1 || len(repositories[0].Sessions) != 2 || !repositories[0].LastSeenAt.Equal(materialAt) {
		t.Fatalf("material save did not include latest in-memory heartbeat state: %#v", repositories)
	}
}

func TestCatalogSessionEndPersistsImmediately(t *testing.T) {
	catalog, path := heartbeatTestCatalog(t)
	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	observation := heartbeatObservation(123)
	if _, err := catalog.Observe([]Observation{observation}, startedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe([]Observation{observation}, startedAt.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	beforeEnd := readCatalogBytes(t, path)

	endedAt := startedAt.Add(4 * time.Second)
	if _, err := catalog.Observe(nil, endedAt); err != nil {
		t.Fatal(err)
	}
	if got := readCatalogBytes(t, path); bytes.Equal(beforeEnd, got) {
		t.Fatal("session end did not persist immediately")
	}

	reloaded, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	session := reloaded.Repositories()[0].Sessions[0]
	if session.Active || session.EndedAt == nil || !session.EndedAt.Equal(endedAt) {
		t.Fatalf("session end was not durable: %#v", session)
	}
}

func TestCatalogFlushPersistsPendingHeartbeatAndIsIdempotentWhenClean(t *testing.T) {
	catalog, path := heartbeatTestCatalog(t)
	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	observation := heartbeatObservation(123)
	if _, err := catalog.Observe([]Observation{observation}, startedAt); err != nil {
		t.Fatal(err)
	}
	initial := readCatalogBytes(t, path)

	heartbeatAt := startedAt.Add(2 * time.Second)
	if _, err := catalog.Observe([]Observation{observation}, heartbeatAt); err != nil {
		t.Fatal(err)
	}
	if !catalog.heartbeatDirty {
		t.Fatal("expected coalesced heartbeat to remain dirty before Flush")
	}
	if err := catalog.Flush(); err != nil {
		t.Fatal(err)
	}
	if catalog.heartbeatDirty {
		t.Fatal("Flush did not clear heartbeat dirtiness")
	}
	flushed := readCatalogBytes(t, path)
	if bytes.Equal(initial, flushed) {
		t.Fatal("Flush did not persist pending heartbeat state")
	}
	persistedAt := catalog.lastPersistedAt

	if err := catalog.Flush(); err != nil {
		t.Fatal(err)
	}
	if !catalog.lastPersistedAt.Equal(persistedAt) {
		t.Fatal("clean Flush unexpectedly performed another persistence cycle")
	}

	reloaded, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Repositories()
	if len(got) != 1 || !got[0].LastSeenAt.Equal(heartbeatAt) {
		t.Fatalf("flushed heartbeat was not durable: %#v", got)
	}
}

func TestCatalogFirstHeartbeatAfterReopenMayPersistImmediately(t *testing.T) {
	catalog, path := heartbeatTestCatalog(t)
	startedAt := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	observation := heartbeatObservation(123)
	if _, err := catalog.Observe([]Observation{observation}, startedAt); err != nil {
		t.Fatal(err)
	}
	before := readCatalogBytes(t, path)

	reopened, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	heartbeatAt := startedAt.Add(2 * time.Second)
	if _, err := reopened.Observe([]Observation{observation}, heartbeatAt); err != nil {
		t.Fatal(err)
	}
	if got := readCatalogBytes(t, path); bytes.Equal(before, got) {
		t.Fatal("first heartbeat after reopen should favor freshness and may persist immediately")
	}
}
