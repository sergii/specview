package federation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sergii/specview/internal/controlplane"
)

func TestDecodeSnapshotAcceptsValidatedV3RepositoryControlPlane(t *testing.T) {
	snapshot := v3SnapshotFixture(t)
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSnapshot(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != SnapshotSchemaVersionV3 || decoded.Instances[0].ControlPlane == nil {
		t.Fatalf("unexpected v3 snapshot: %#v", decoded)
	}
	if decoded.Instances[0].ControlPlane.RepositoryID != decoded.Instances[0].SourceRepositoryID {
		t.Fatalf("repository authority changed: %#v", decoded.Instances[0].ControlPlane)
	}
}

func TestSnapshotV3RejectsMissingOrMismatchedRepositoryControlPlane(t *testing.T) {
	for name, mutate, want := range map[string]struct {
		mutate func(*HostSnapshot)
		want   string
	}{
		"missing": {
			mutate: func(snapshot *HostSnapshot) { snapshot.Instances[0].ControlPlane = nil },
			want:   "requires control_plane",
		},
		"host": {
			mutate: func(snapshot *HostSnapshot) { snapshot.Instances[0].ControlPlane.Host = "other-host" },
			want:   "does not match hostname",
		},
		"repository-id": {
			mutate: func(snapshot *HostSnapshot) { snapshot.Instances[0].ControlPlane.RepositoryID = "repo:other" },
			want:   "does not match source repository ID",
		},
		"repository-name": {
			mutate: func(snapshot *HostSnapshot) { snapshot.Instances[0].ControlPlane.RepositoryName = "other/repo" },
			want:   "does not match instance name",
		},
		"schema": {
			mutate: func(snapshot *HostSnapshot) { snapshot.Instances[0].ControlPlane.SchemaVersion++ },
			want:   "unsupported repository control-plane schema version",
		},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := v3SnapshotFixture(t)
			mutate(&snapshot)
			if err := snapshot.Validate(); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("Validate error = %v, want %q", err, want)
			}
		})
	}
}

func TestOlderSnapshotVersionsRejectRepositoryControlPlaneLeakage(t *testing.T) {
	snapshot := v3SnapshotFixture(t)
	snapshot.SchemaVersion = SnapshotSchemaVersionV2
	if err := snapshot.Validate(); err == nil || !strings.Contains(err.Error(), "must not contain repository control_plane") {
		t.Fatalf("v2 repository control-plane leakage error = %v", err)
	}

	snapshot = v3SnapshotFixture(t)
	snapshot.SchemaVersion = SnapshotSchemaVersion
	snapshot.ControlPlane = nil
	if err := snapshot.Validate(); err == nil || !strings.Contains(err.Error(), "must not contain repository control_plane") {
		t.Fatalf("v1 repository control-plane leakage error = %v", err)
	}
}

func v3SnapshotFixture(t *testing.T) HostSnapshot {
	t.Helper()
	snapshot := loadSnapshotFixture(t, "v1-laptop.json")
	snapshot.SchemaVersion = SnapshotSchemaVersionV3
	snapshot.ControlPlane = &controlplane.GetHostControlPlaneResult{
		SchemaVersion: controlplane.SchemaVersion,
		Host:          snapshot.Hostname,
		Attention:     []controlplane.HostAttentionSummary{},
	}
	for i := range snapshot.Instances {
		instance := &snapshot.Instances[i]
		instance.ControlPlane = &controlplane.GetRepositoryControlPlaneResult{
			SchemaVersion:  controlplane.SchemaVersion,
			Host:           snapshot.Hostname,
			RepositoryID:   instance.SourceRepositoryID,
			RepositoryName: instance.Name,
			Intent:         controlplane.RepositoryIntentSummary{Total: 2, InProgress: 1, Done: 1},
			Evidence:       controlplane.RepositoryEvidenceOverviewSummary{Total: 1, Passed: 1},
			Acceptance:     controlplane.RepositoryAcceptanceOverviewSummary{Configured: true, Accepted: 1},
		}
	}
	return snapshot
}
