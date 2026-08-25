package federationruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
)

type snapshotBuilderStub struct {
	snapshot federation.HostSnapshot
	err      error
}

func (s snapshotBuilderStub) Build(context.Context) (federation.HostSnapshot, error) {
	return s.snapshot, s.err
}

type fixedAggregator struct {
	at time.Time
}

func (a fixedAggregator) Aggregate(snapshots ...federation.HostSnapshot) (federation.Projection, error) {
	projection, err := federation.NewAggregator().Aggregate(snapshots...)
	if err != nil {
		return federation.Projection{}, err
	}
	projection.GeneratedAt = a.at.UTC()
	return projection, nil
}

func TestProjectionV3ContractKeepsV1PeerWithoutInventingControlPlane(t *testing.T) {
	root := t.TempDir()
	registryPath := filepath.Join(root, "federation-peers.json")
	observationDir := filepath.Join(root, "federation", "peers")
	registry, err := federationpeers.OpenRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	devbox := federationpeers.Peer{
		Name:              "devbox",
		URL:               "https://devbox.example.ts.net/v1/federation/snapshot",
		ExpectedHostID:    "host:22222222-2222-4222-9222-222222222222",
		StaleAfterSeconds: 300,
	}
	never := federationpeers.Peer{
		Name:              "never",
		URL:               "https://never.example.ts.net/v1/federation/snapshot",
		ExpectedHostID:    "host:33333333-3333-4333-9333-333333333333",
		StaleAfterSeconds: 300,
	}
	if err := registry.Add(devbox); err != nil {
		t.Fatal(err)
	}
	if err := registry.Add(never); err != nil {
		t.Fatal(err)
	}

	store := federationpeers.NewObservationStore(observationDir)
	devboxSnapshot := federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersion,
		HostID:        devbox.ExpectedHostID,
		Hostname:      "devbox-01",
		ObservedAt:    time.Date(2026, 8, 23, 20, 0, 5, 0, time.UTC),
		Instances:     []federation.RepositoryInstance{},
	}
	if _, err := store.RecordSuccess(devbox, devboxSnapshot, time.Date(2026, 8, 23, 20, 0, 10, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordFailure(devbox, errors.New("offline"), time.Date(2026, 8, 23, 20, 0, 20, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	local := federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersionV2,
		HostID:        "host:11111111-1111-4111-9111-111111111111",
		Hostname:      "sergii-macbook",
		ObservedAt:    time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC),
		ControlPlane: &controlplane.GetHostControlPlaneResult{
			SchemaVersion: controlplane.SchemaVersion,
			Host:          "sergii-macbook",
			Intent: controlplane.HostIntentSummary{
				ManagedRepositories: 1,
				WorkItems:           2,
				InProgress:          1,
			},
			Execution: controlplane.HostExecutionSummary{
				ActiveSessions:     1,
				ActiveRepositories: 1,
			},
			Attention: []controlplane.HostAttentionSummary{},
		},
		Instances: []federation.RepositoryInstance{},
	}
	builder, err := NewProjectionBuilder(snapshotBuilderStub{snapshot: local}, registryPath, store)
	if err != nil {
		t.Fatal(err)
	}
	fixedTime := time.Date(2026, 8, 23, 20, 0, 30, 0, time.UTC)
	builder.now = func() time.Time { return fixedTime }
	builder.aggregator = fixedAggregator{at: fixedTime}

	actual, err := builder.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if actual.Hosts[0].ControlPlane == nil || actual.Hosts[0].ControlPlane.Execution.ActiveSessions != 1 {
		t.Fatalf("local v2 control plane missing: %#v", actual.Hosts[0])
	}
	if actual.Hosts[1].ControlPlane != nil {
		t.Fatalf("v1 peer invented control-plane facts: %#v", actual.Hosts[1])
	}

	var expected Projection
	if err := json.Unmarshal(readRuntimeFixture(t, "v3-status.json"), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		actualJSON, _ := json.MarshalIndent(actual, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Fatalf("multi-host projection mismatch\nactual:\n%s\nexpected:\n%s", actualJSON, expectedJSON)
	}
}

func TestProjectionKeepsCachedUnreachableV2ControlPlaneFacts(t *testing.T) {
	root := t.TempDir()
	registryPath := filepath.Join(root, "federation-peers.json")
	store := federationpeers.NewObservationStore(filepath.Join(root, "federation", "peers"))
	registry, err := federationpeers.OpenRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	laptop := loadFederationFixture(t, "v1-laptop.json")
	laptop.SchemaVersion = federation.SnapshotSchemaVersionV2
	laptop.ControlPlane = fixtureControlPlane(laptop.Hostname, 1, 0, 0)
	devboxSnapshot := loadFederationFixture(t, "v1-devbox.json")
	devboxSnapshot.SchemaVersion = federation.SnapshotSchemaVersionV2
	devboxSnapshot.ControlPlane = fixtureControlPlane(devboxSnapshot.Hostname, 2, 1, 1)
	devboxSnapshot.ControlPlane.Attention = []controlplane.HostAttentionSummary{{
		RepositoryID:   devboxSnapshot.Instances[0].SourceRepositoryID,
		RepositoryName: devboxSnapshot.Instances[0].Name,
		LastSeenAt:     devboxSnapshot.ObservedAt,
		Signals:        []string{"1 failed Evidence record", "1 blocked Acceptance item"},
	}}
	peer := federationpeers.Peer{
		Name:              "devbox",
		URL:               "https://devbox.example.ts.net/v2/federation/snapshot",
		ExpectedHostID:    devboxSnapshot.HostID,
		StaleAfterSeconds: 300,
	}
	if err := registry.Add(peer); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordSuccess(peer, devboxSnapshot, time.Date(2026, 8, 23, 20, 0, 10, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordFailure(peer, errors.New("dial failed"), time.Date(2026, 8, 23, 20, 0, 20, 0, time.UTC)); err != nil {
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
	if len(projection.Hosts) != 2 || projection.Hosts[1].Freshness != federationpeers.FreshnessUnreachable || !projection.Hosts[1].HasSnapshot {
		t.Fatalf("unexpected remote Host status: %#v", projection.Hosts)
	}
	remoteControlPlane := projection.Hosts[1].ControlPlane
	if remoteControlPlane == nil {
		t.Fatalf("cached unreachable v2 peer lost control-plane facts: %#v", projection.Hosts[1])
	}
	if remoteControlPlane.Execution.ActiveSessions != 2 || remoteControlPlane.Evidence.Failed != 1 || remoteControlPlane.Acceptance.Blocked != 1 || len(remoteControlPlane.Attention) != 1 {
		t.Fatalf("unexpected cached remote control plane: %#v", remoteControlPlane)
	}
	if len(projection.Federation.Repositories) != 1 || len(projection.Federation.Repositories[0].Instances) != 2 {
		t.Fatalf("cached remote repository facts disappeared: %#v", projection.Federation.Repositories)
	}
	instances := projection.Federation.Repositories[0].Instances
	sessionCount := len(instances[0].Sessions) + len(instances[1].Sessions)
	if sessionCount != 3 {
		t.Fatalf("federated session count = %d, want 3", sessionCount)
	}
	if projection.Hosts[1].ObservedAt == nil || !projection.Hosts[1].ObservedAt.Equal(devboxSnapshot.ObservedAt) {
		t.Fatalf("source observed_at was rewritten: %#v", projection.Hosts[1])
	}
}

func TestProjectionShowsNeverRetrievedPeerWithoutInventingFacts(t *testing.T) {
	root := t.TempDir()
	registryPath := filepath.Join(root, "federation-peers.json")
	store := federationpeers.NewObservationStore(filepath.Join(root, "federation", "peers"))
	registry, err := federationpeers.OpenRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	peer := federationpeers.Peer{
		Name:              "never",
		URL:               "https://never.example.ts.net/v2/federation/snapshot",
		ExpectedHostID:    "host:33333333-3333-4333-9333-333333333333",
		StaleAfterSeconds: 300,
	}
	if err := registry.Add(peer); err != nil {
		t.Fatal(err)
	}
	local := federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersionV2,
		HostID:        "host:11111111-1111-4111-9111-111111111111",
		Hostname:      "laptop",
		ObservedAt:    time.Now().UTC(),
		ControlPlane:  fixtureControlPlane("laptop", 0, 0, 0),
		Instances:     []federation.RepositoryInstance{},
	}
	builder, err := NewProjectionBuilder(snapshotBuilderStub{snapshot: local}, registryPath, store)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := builder.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Hosts) != 2 || projection.Hosts[1].Freshness != federationpeers.FreshnessNeverRetrieved || projection.Hosts[1].HasSnapshot || projection.Hosts[1].ControlPlane != nil {
		t.Fatalf("unexpected never-retrieved status: %#v", projection.Hosts)
	}
	if len(projection.Federation.Hosts) != 1 {
		t.Fatalf("never-retrieved peer invented federation facts: %#v", projection.Federation.Hosts)
	}
}

func TestProjectionFailsWhenLocalSnapshotFails(t *testing.T) {
	builder, err := NewProjectionBuilder(snapshotBuilderStub{err: errors.New("local failed")}, "", federationpeers.NewObservationStore(""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := builder.Build(context.Background()); err == nil {
		t.Fatal("expected local snapshot failure")
	}
}

func fixtureControlPlane(host string, activeSessions, failedEvidence, blockedAcceptance int) *controlplane.GetHostControlPlaneResult {
	return &controlplane.GetHostControlPlaneResult{
		SchemaVersion: controlplane.SchemaVersion,
		Host:          host,
		Execution: controlplane.HostExecutionSummary{
			ActiveSessions: activeSessions,
		},
		Evidence: controlplane.HostEvidenceSummary{
			Failed: failedEvidence,
		},
		Acceptance: controlplane.HostAcceptanceSummary{
			Blocked: blockedAcceptance,
		},
		Attention: []controlplane.HostAttentionSummary{},
	}
}

func loadFederationFixture(t *testing.T, name string) federation.HostSnapshot {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "testdata", "contracts", "federation", name))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := federation.DecodeSnapshot(data)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func readRuntimeFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "testdata", "contracts", "federation-runtime", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
