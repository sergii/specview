package federation

import "testing"

func TestAggregatePreservesRepositoryControlPlaneOnlyFromV3Source(t *testing.T) {
	laptop := v3SnapshotFixture(t)
	devbox := loadSnapshotFixture(t, "v1-devbox.json")

	projection, err := NewAggregator().Aggregate(laptop, devbox)
	if err != nil {
		t.Fatal(err)
	}
	if projection.SchemaVersion != ProjectionSchemaVersion {
		t.Fatalf("nested federation projection schema = %d, want %d", projection.SchemaVersion, ProjectionSchemaVersion)
	}
	if len(projection.Repositories) != 1 || len(projection.Repositories[0].Instances) != 2 {
		t.Fatalf("unexpected correlated repository group: %#v", projection.Repositories)
	}

	withControlPlane := 0
	withoutControlPlane := 0
	for _, instance := range projection.Repositories[0].Instances {
		if instance.HostID == laptop.HostID {
			withControlPlane++
			if instance.ControlPlane == nil {
				t.Fatalf("v3 source repository control plane was lost: %#v", instance)
			}
			if instance.ControlPlane.RepositoryID != instance.SourceRepositoryID || instance.ControlPlane.RepositoryName != instance.Name {
				t.Fatalf("v3 source repository authority changed: %#v", instance.ControlPlane)
			}
			continue
		}
		withoutControlPlane++
		if instance.ControlPlane != nil {
			t.Fatalf("v1 source repository control plane was invented: %#v", instance.ControlPlane)
		}
	}
	if withControlPlane != 1 || withoutControlPlane != 1 {
		t.Fatalf("unexpected source attribution counts: with=%d without=%d", withControlPlane, withoutControlPlane)
	}
}
