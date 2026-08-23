package federationruntime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
)

type materialRefresher struct {
	status federationpeers.PeerStatus
	err    error
}

func (r *materialRefresher) Refresh(_ context.Context, peer federationpeers.Peer) (federationpeers.PeerStatus, error) {
	status := r.status
	status.Peer = peer
	return status, r.err
}

func TestPollerNotifiesOnlyOnMaterialPeerChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "federation-peers.json")
	registry, err := federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	peer := testPeer("alpha", "host:11111111-1111-4111-9111-111111111111")
	if err := registry.Add(peer); err != nil {
		t.Fatal(err)
	}

	snapshot := federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersion,
		HostID:        peer.ExpectedHostID,
		Hostname:      "alpha-host",
		ObservedAt:    time.Date(2026, 8, 23, 21, 0, 0, 0, time.UTC),
		Instances:     []federation.RepositoryInstance{},
	}
	refresher := &materialRefresher{
		status: federationpeers.PeerStatus{
			Freshness: federationpeers.FreshnessFresh,
			Snapshot:  &snapshot,
		},
	}
	changes := 0
	poller, err := NewPoller(path, refresher, time.Second, func() { changes++ })
	if err != nil {
		t.Fatal(err)
	}

	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if changes != 1 {
		t.Fatalf("identical refresh cycles produced %d callbacks, want 1 initial callback", changes)
	}

	refresher.status.Freshness = federationpeers.FreshnessUnreachable
	refresher.status.LastError = "offline"
	refresher.err = errors.New("offline")
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if changes != 2 {
		t.Fatalf("material freshness/error change produced %d callbacks, want 2", changes)
	}
}
