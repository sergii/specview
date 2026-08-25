package federationruntime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
)

func TestProjectionV3PreservesLocalAndCachedRepositoryControlPlaneAuthority(t *testing.T) {
	root := t.TempDir()
	registryPath := filepath.Join(root, "federation-peers.json")
	store := federationpeers.NewObservationStore(filepath.Join(root, "federation", "peers"))
	registry, err := federationpeers.OpenRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	laptop := loadFederationFixture(t, "v1-laptop.json")
	upgradeSnapshotToV3(laptop.Hostname, &laptop, 2, 1, 0)
	devbox := loadFederationFixture(t, "v1-devbox.json")
	upgradeSnapshotToV3(devbox.Hostname, &devbox, 7, 3, 2)

	peer := federationpeers.Peer{
		Name:              "devbox-v3",
		URL:               "https://devbox.example.ts.net/v3/federation/snapshot",
		ExpectedHostID:    devbox.HostID,
		StaleAfterSeconds: 300,
	}
	if err := registry.Add(peer); err != nil {
		t.Fatal(err)
	}
	successAt := time.Date(2026, 8, 23, 20, 0, 10, 0, time.UTC)
	if _, err := store.RecordSuccess(peer, devbox, successAt); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordFailure(peer, errors.New("offline"), successAt.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}

	builder, err := NewProjectionBuilder(snapshotBuilderStub{snapshot: laptop}, registryPath, store)
	if err != nil {
		t.Fatal(err)
	}
	builder.now = func() time.Time { return time.Date(2026, 8, 23, 20, 0, 30, 0, time.UTC) }

	projection, err := builder.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if projection.SchemaVersion != ProjectionSchemaVersion || projection.SchemaVersion != 3 {
		t.Fatalf("runtime projection schema = %d, want v3", projection.SchemaVersion)
	}
	if projection.Federation.SchemaVersion != federation.ProjectionSchemaVersion {
		t.Fatalf("nested H20 projection schema changed: %d", projection.Federation.SchemaVersion)
	}
	if len(projection.Hosts) != 2 || projection.Hosts[1].Freshness != federationpeers.FreshnessUnreachable {
		t.Fatalf("cached peer freshness/source attribution changed: %#v", projection.Hosts)
	}
	if len(projection.Federation.Repositories) != 1 || len(projection.Federation.Repositories[0].Instances) != 2 {
		t.Fatalf("unexpected repository correlation: %#v", projection.Federation.Repositories)
	}

	seenLocal := false
	seenRemote := false
	for _, instance := range projection.Federation.Repositories[0].Instances {
		if instance.ControlPlane == nil {
			t.Fatalf("v3 repository control plane lost from runtime projection: %#v", instance)
		}
		if instance.ControlPlane.RepositoryID != instance.SourceRepositoryID || instance.ControlPlane.RepositoryName != instance.Name {
			t.Fatalf("runtime projection rewrote repository authority: %#v", instance.ControlPlane)
		}
		switch instance.HostID {
		case laptop.HostID:
			seenLocal = true
			if instance.ControlPlane.Intent.Total != 2 || instance.ControlPlane.Evidence.Failed != 1 || instance.ControlPlane.Acceptance.Blocked != 0 {
				t.Fatalf("local repository control plane changed: %#v", instance.ControlPlane)
			}
		case devbox.HostID:
			seenRemote = true
			if instance.ControlPlane.Intent.Total != 7 || instance.ControlPlane.Evidence.Failed != 3 || instance.ControlPlane.Acceptance.Blocked != 2 {
				t.Fatalf("cached repository control plane was recomputed or changed: %#v", instance.ControlPlane)
			}
		}
	}
	if !seenLocal || !seenRemote {
		t.Fatalf("missing source repository instances: local=%v remote=%v", seenLocal, seenRemote)
	}
}

func upgradeSnapshotToV3(host string, snapshot *federation.HostSnapshot, workItems, failedEvidence, blockedAcceptance int) {
	snapshot.SchemaVersion = federation.SnapshotSchemaVersionV3
	snapshot.ControlPlane = fixtureControlPlane(host, 0, failedEvidence, blockedAcceptance)
	for i := range snapshot.Instances {
		instance := &snapshot.Instances[i]
		instance.ControlPlane = &controlplane.GetRepositoryControlPlaneResult{
			SchemaVersion:  controlplane.SchemaVersion,
			Host:           host,
			RepositoryID:   instance.SourceRepositoryID,
			RepositoryName: instance.Name,
			Intent:         controlplane.RepositoryIntentSummary{Total: workItems, InProgress: workItems},
			Evidence:       controlplane.RepositoryEvidenceOverviewSummary{Total: failedEvidence, Failed: failedEvidence},
			Acceptance:     controlplane.RepositoryAcceptanceOverviewSummary{Configured: true, Blocked: blockedAcceptance},
		}
	}
}
