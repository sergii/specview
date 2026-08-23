package federation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/sergii/specview/internal/identity"
)

func TestAggregateTwoHostContractFixtures(t *testing.T) {
	laptop := loadSnapshotFixture(t, "v1-laptop.json")
	devbox := loadSnapshotFixture(t, "v1-devbox.json")

	aggregator := NewAggregator()
	aggregator.now = func() time.Time {
		return time.Date(2026, 8, 23, 20, 0, 10, 0, time.UTC)
	}
	projection, err := aggregator.Aggregate(laptop, devbox)
	if err != nil {
		t.Fatal(err)
	}

	var expected Projection
	if err := json.Unmarshal(readFederationFixture(t, "v1-projection.json"), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(projection, expected) {
		actualJSON, _ := json.MarshalIndent(projection, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Fatalf("federation projection mismatch\nactual:\n%s\nexpected:\n%s", actualJSON, expectedJSON)
	}

	if len(projection.Repositories) != 1 || len(projection.Repositories[0].Instances) != 2 {
		t.Fatalf("unexpected repository grouping: %#v", projection.Repositories)
	}
	if got := len(projection.Repositories[0].Instances[0].Sessions) + len(projection.Repositories[0].Instances[1].Sessions); got != 3 {
		t.Fatalf("federated session count = %d, want 3", got)
	}
}

func TestAggregateKeepsNameOnlyCorrelationSeparate(t *testing.T) {
	left := snapshotForFingerprint(t,
		"host:11111111-1111-4111-9111-111111111111",
		"laptop",
		"/work/laptop/api",
		"repo-left",
		identity.RepositoryFingerprint{Name: "team/api"},
	)
	right := snapshotForFingerprint(t,
		"host:22222222-2222-4222-9222-222222222222",
		"devbox",
		"/work/devbox/api",
		"repo-right",
		identity.RepositoryFingerprint{Name: "team/api"},
	)

	projection, err := NewAggregator().Aggregate(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Repositories) != 2 {
		t.Fatalf("same-name-only repositories must remain separate: %#v", projection.Repositories)
	}
	if len(projection.Issues) != 1 || projection.Issues[0].Outcome != identity.CorrelationAmbiguous {
		t.Fatalf("expected one ambiguous correlation issue, got %#v", projection.Issues)
	}
}

func TestAggregateKeepsConflictingExplicitIdentitySeparate(t *testing.T) {
	left := snapshotForFingerprint(t,
		"host:11111111-1111-4111-9111-111111111111",
		"laptop",
		"/work/laptop/specview",
		"repo-left",
		identity.RepositoryFingerprint{
			ExplicitID: "specview:sergii/specview",
			Name:       "sergii/specview",
			GitRemote:  "git@github.com:sergii/specview.git",
		},
	)
	right := snapshotForFingerprint(t,
		"host:22222222-2222-4222-9222-222222222222",
		"devbox",
		"/work/devbox/specview",
		"repo-right",
		identity.RepositoryFingerprint{
			ExplicitID: "specview:sergii/specview",
			Name:       "sergii/specview",
			GitRemote:  "git@github.com:other/specview.git",
		},
	)

	projection, err := NewAggregator().Aggregate(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Repositories) != 2 {
		t.Fatalf("conflicting repositories must remain separate: %#v", projection.Repositories)
	}
	if len(projection.Issues) != 1 || projection.Issues[0].Outcome != identity.CorrelationConflict {
		t.Fatalf("expected one conflict issue, got %#v", projection.Issues)
	}
}

func TestAggregateDoesNotBridgeDistinctGroupsTransitively(t *testing.T) {
	shared := identity.RepositoryFingerprint{Name: "team/api", GitRemote: "git@github.com:team/api.git"}
	leftFingerprint := shared
	leftFingerprint.ExplicitID = "specview:team/api-a"
	rightFingerprint := shared
	rightFingerprint.ExplicitID = "specview:team/api-b"

	left := snapshotForFingerprint(t,
		"host:11111111-1111-4111-9111-111111111111",
		"host-a",
		"/work/a/api",
		"repo-a",
		leftFingerprint,
	)
	right := snapshotForFingerprint(t,
		"host:22222222-2222-4222-9222-222222222222",
		"host-b",
		"/work/b/api",
		"repo-b",
		rightFingerprint,
	)
	bridge := snapshotForFingerprint(t,
		"host:33333333-3333-4333-8333-333333333333",
		"host-c",
		"/work/c/api",
		"repo-c",
		shared,
	)

	projection, err := NewAggregator().Aggregate(left, right, bridge)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Repositories) != 3 {
		t.Fatalf("bridge candidate must not merge distinct groups: %#v", projection.Repositories)
	}
	ambiguous := 0
	for _, issue := range projection.Issues {
		if issue.Outcome == identity.CorrelationAmbiguous {
			ambiguous++
		}
	}
	if ambiguous != 2 {
		t.Fatalf("bridge candidate should surface two ambiguity issues, got %#v", projection.Issues)
	}
}

func TestAggregateUsesNewestSnapshotPerHost(t *testing.T) {
	current := loadSnapshotFixture(t, "v1-laptop.json")
	old := current
	old.ObservedAt = current.ObservedAt.Add(-time.Minute)
	old.Hostname = "old-hostname"
	old.Instances = append([]RepositoryInstance(nil), current.Instances...)
	old.Instances[0].Agents = []string{"Old Agent"}
	old.Instances[0].Sessions = []Session{{ID: "old-session", Adapter: "old", Agent: "Old Agent", CWD: old.Instances[0].Root}}

	projection, err := NewAggregator().Aggregate(current, old)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Hosts) != 1 || projection.Hosts[0].Hostname != "sergii-macbook" {
		t.Fatalf("newest Host snapshot was not selected: %#v", projection.Hosts)
	}
	if len(projection.Repositories) != 1 || len(projection.Repositories[0].Instances) != 1 {
		t.Fatalf("old Host snapshot leaked into current projection: %#v", projection.Repositories)
	}
	instance := projection.Repositories[0].Instances[0]
	if len(instance.Sessions) != 1 || instance.Sessions[0].ID != "codex-laptop-1" {
		t.Fatalf("old sessions leaked into current projection: %#v", instance.Sessions)
	}
}

func TestAggregateRejectsConflictingSnapshotsAtSameObservationTime(t *testing.T) {
	left := loadSnapshotFixture(t, "v1-laptop.json")
	right := left
	right.Hostname = "different-hostname"
	if _, err := NewAggregator().Aggregate(left, right); err == nil {
		t.Fatal("expected conflicting same-time Host snapshots to fail")
	}
}

func snapshotForFingerprint(t *testing.T, hostID, hostname, root, repositoryID string, fingerprint identity.RepositoryFingerprint) HostSnapshot {
	t.Helper()
	instanceID, err := identity.RepositoryInstanceID(hostID, root)
	if err != nil {
		t.Fatal(err)
	}
	return HostSnapshot{
		SchemaVersion: SnapshotSchemaVersion,
		HostID:        hostID,
		Hostname:      hostname,
		ObservedAt:    time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC),
		Instances: []RepositoryInstance{{
			InstanceID:         instanceID,
			SourceRepositoryID: repositoryID,
			Name:               fingerprint.Name,
			Root:               root,
			Fingerprint:        fingerprint,
			Sessions:           []Session{},
			Worktrees:          []Worktree{},
		}},
	}
}

func loadSnapshotFixture(t *testing.T, name string) HostSnapshot {
	t.Helper()
	snapshot, err := DecodeSnapshot(readFederationFixture(t, name))
	if err != nil {
		t.Fatalf("decode federation fixture %s: %v", name, err)
	}
	return snapshot
}

func readFederationFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve federation test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "testdata", "contracts", "federation", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
