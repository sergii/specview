package federationpeers

import (
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/identity"
)

func TestObservationStoreRoundTripsV3RepositoryControlPlaneWithoutEnvelopeMigration(t *testing.T) {
	root := "/srv/specview"
	instanceID, err := identity.RepositoryInstanceID(testHostID, root)
	if err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	snapshot := federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersionV3,
		HostID:        testHostID,
		Hostname:      "devbox-01",
		ObservedAt:    observedAt,
		ControlPlane: &controlplane.GetHostControlPlaneResult{
			SchemaVersion: controlplane.SchemaVersion,
			Host:          "devbox-01",
			Attention:     []controlplane.HostAttentionSummary{},
		},
		Instances: []federation.RepositoryInstance{{
			InstanceID:         instanceID,
			SourceRepositoryID: "repo-remote",
			Name:               "sergii/specview",
			Root:               root,
			Sessions:           []federation.Session{},
			Worktrees:          []federation.Worktree{},
			ControlPlane: &controlplane.GetRepositoryControlPlaneResult{
				SchemaVersion:  controlplane.SchemaVersion,
				Host:           "devbox-01",
				RepositoryID:   "repo-remote",
				RepositoryName: "sergii/specview",
				Intent:         controlplane.RepositoryIntentSummary{Total: 3, InProgress: 2, Done: 1},
				Evidence:       controlplane.RepositoryEvidenceOverviewSummary{Total: 2, Passed: 1, Failed: 1},
				Acceptance:     controlplane.RepositoryAcceptanceOverviewSummary{Configured: true, Accepted: 1, Blocked: 1},
			},
		}},
	}
	peer := testPeer("https://devbox.example.test/v3/federation/snapshot")
	store := NewObservationStore(t.TempDir())
	retrievedAt := observedAt.Add(10 * time.Second)

	recorded, err := store.RecordSuccess(peer, snapshot, retrievedAt)
	if err != nil {
		t.Fatal(err)
	}
	if recorded.Version != ObservationVersion {
		t.Fatalf("observation envelope version = %d, want %d", recorded.Version, ObservationVersion)
	}
	loaded, err := store.Load(peer.Name)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != ObservationVersion || loaded.Snapshot == nil || loaded.Snapshot.SchemaVersion != federation.SnapshotSchemaVersionV3 {
		t.Fatalf("unexpected persisted v3 observation: %#v", loaded)
	}
	if len(loaded.Snapshot.Instances) != 1 || loaded.Snapshot.Instances[0].ControlPlane == nil {
		t.Fatalf("repository control plane lost from persisted snapshot: %#v", loaded.Snapshot)
	}
	controlPlane := loaded.Snapshot.Instances[0].ControlPlane
	if controlPlane.RepositoryID != "repo-remote" || controlPlane.Intent.Total != 3 || controlPlane.Evidence.Failed != 1 || controlPlane.Acceptance.Blocked != 1 {
		t.Fatalf("repository control-plane facts changed after round trip: %#v", controlPlane)
	}
}
