package hoststate

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type idleScanner struct{}

func (idleScanner) Scan() ([]Observation, error) {
	return nil, nil
}

func TestRuntimeRunFlushesPendingHeartbeatOnCancellation(t *testing.T) {
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
	if got := readCatalogBytes(t, path); !bytes.Equal(initial, got) {
		t.Fatal("precondition failed: pending heartbeat was already persisted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	NewRuntime(catalog, idleScanner{}, time.Hour, nil).Run(ctx)

	if got := readCatalogBytes(t, path); bytes.Equal(initial, got) {
		t.Fatal("runtime cancellation did not flush pending heartbeat state")
	}
	reloaded, err := openCatalog(path, catalog.detect)
	if err != nil {
		t.Fatal(err)
	}
	repositories := reloaded.Repositories()
	if len(repositories) != 1 || !repositories[0].LastSeenAt.Equal(heartbeatAt) {
		t.Fatalf("runtime shutdown did not persist latest heartbeat: %#v", repositories)
	}
}
