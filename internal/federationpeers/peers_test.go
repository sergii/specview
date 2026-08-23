package federationpeers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationhttp"
)

const testHostID = "host:22222222-2222-4222-9222-222222222222"

func TestPeerRegistryContractFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "contracts", "peers", "v1.json")
	registry, err := OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	peers := registry.List()
	if len(peers) != 2 {
		t.Fatalf("peers = %d, want 2", len(peers))
	}
	devbox, ok := registry.Find("devbox")
	if !ok {
		t.Fatal("devbox peer missing")
	}
	if devbox.ExpectedHostID != testHostID || devbox.StaleAfterSeconds != 300 || devbox.Credentials != nil {
		t.Fatalf("unexpected devbox peer: %#v", devbox)
	}
	cloud, ok := registry.Find("cloud-devbox")
	if !ok || cloud.Credentials == nil || cloud.Credentials.Type != "env_headers" {
		t.Fatalf("unexpected cloud peer: %#v", cloud)
	}
	if cloud.Credentials.Headers["CF-Access-Client-Secret"] != "SPECVIEW_CLOUDFLARE_CLIENT_SECRET" {
		t.Fatalf("credential ref mismatch: %#v", cloud.Credentials)
	}
}

func TestRemoteObservationContractFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "contracts", "remote-observation", "v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := decodeObservation(data)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Peer != "devbox" || observation.Snapshot == nil || observation.Snapshot.HostID != testHostID {
		t.Fatalf("unexpected observation: %#v", observation)
	}
	if !observation.Snapshot.ObservedAt.Equal(time.Date(2026, 8, 23, 20, 29, 45, 0, time.UTC)) {
		t.Fatalf("observed_at changed: %s", observation.Snapshot.ObservedAt)
	}
}

func TestRegistryPersistsReferencesWithoutSecretValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "federation-peers.json")
	registry, err := OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	peer := testPeer("https://devbox.example.test/v1/federation/snapshot")
	peer.Credentials = &CredentialRef{Type: "env_headers", Headers: map[string]string{
		"Authorization": "SPECVIEW_TEST_TOKEN",
	}}
	if err := registry.Add(peer); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatal("secret value leaked into peer registry")
	}
	if !strings.Contains(string(data), "SPECVIEW_TEST_TOKEN") {
		t.Fatal("environment variable reference missing")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("registry permissions = %o, want private", info.Mode().Perm())
	}
}

func TestResolveCredentialHeadersFailsBeforeSecretCanLeak(t *testing.T) {
	const envName = "SPECVIEW_MISSING_SECRET_FOR_TEST"
	_ = os.Unsetenv(envName)
	credentials := &CredentialRef{Type: "env_headers", Headers: map[string]string{"Authorization": envName}}
	_, err := ResolveCredentialHeaders(credentials)
	if err == nil || !strings.Contains(err.Error(), envName) {
		t.Fatalf("expected missing environment variable error, got %v", err)
	}
	if strings.Contains(err.Error(), "super-secret-value") {
		t.Fatal("secret appeared in error")
	}
}

func TestObservationFailurePreservesLastValidSnapshot(t *testing.T) {
	store := NewObservationStore(t.TempDir())
	peer := testPeer("https://devbox.example.test/v1/federation/snapshot")
	snapshot := testSnapshot(time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC))
	successAt := time.Date(2026, 8, 23, 20, 0, 10, 0, time.UTC)
	if _, err := store.RecordSuccess(peer, snapshot, successAt); err != nil {
		t.Fatal(err)
	}
	failureAt := successAt.Add(time.Minute)
	observation, err := store.RecordFailure(peer, errors.New("network unavailable"), failureAt)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Snapshot == nil || observation.Snapshot.HostID != snapshot.HostID {
		t.Fatal("last valid snapshot was lost")
	}
	if observation.RetrievedAt == nil || !observation.RetrievedAt.Equal(successAt) {
		t.Fatalf("retrieved_at changed after failure: %v", observation.RetrievedAt)
	}
	status := ProjectStatus(peer, observation, failureAt)
	if status.Freshness != FreshnessUnreachable {
		t.Fatalf("freshness = %q, want unreachable", status.Freshness)
	}
}

func TestFreshnessProjection(t *testing.T) {
	peer := testPeer("https://devbox.example.test/v1/federation/snapshot")
	peer.StaleAfterSeconds = 300
	base := time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC)
	observation := RemoteObservation{
		Version:       ObservationVersion,
		Peer:          peer.Name,
		RetrievedAt:   timePointer(base.Add(10 * time.Second)),
		LastAttemptAt: timePointer(base.Add(10 * time.Second)),
		LastSuccessAt: timePointer(base.Add(10 * time.Second)),
		Snapshot:      snapshotPointer(testSnapshot(base)),
	}
	if got := ProjectStatus(peer, observation, base.Add(4*time.Minute)).Freshness; got != FreshnessFresh {
		t.Fatalf("fresh status = %q", got)
	}
	if got := ProjectStatus(peer, observation, base.Add(6*time.Minute)).Freshness; got != FreshnessStale {
		t.Fatalf("stale status = %q", got)
	}
	if got := ProjectStatus(peer, emptyObservation(peer.Name), base).Freshness; got != FreshnessNeverRetrieved {
		t.Fatalf("empty status = %q", got)
	}
}

func TestRefresherSendsResolvedHeadersAndPinsHost(t *testing.T) {
	const envName = "SPECVIEW_PEER_AUTH_TEST"
	const secret = "super-secret-value"
	t.Setenv(envName, secret)
	seenSecret := false
	snapshot := testSnapshot(time.Date(2026, 8, 23, 20, 0, 0, 0, time.UTC))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == secret {
			seenSecret = true
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snapshot)
	}))
	defer server.Close()

	peer := testPeer(server.URL + federationhttp.SnapshotPath)
	peer.Credentials = &CredentialRef{Type: "env_headers", Headers: map[string]string{"Authorization": envName}}
	store := NewObservationStore(t.TempDir())
	refresher := NewRefresher(federationhttp.NewClient(), store)
	refresher.now = func() time.Time { return time.Date(2026, 8, 23, 20, 0, 30, 0, time.UTC) }
	status, err := refresher.Refresh(context.Background(), peer)
	if err != nil {
		t.Fatal(err)
	}
	if !seenSecret {
		t.Fatal("resolved credential header was not sent")
	}
	if status.Freshness != FreshnessFresh || status.Snapshot == nil || status.Snapshot.HostID != testHostID {
		t.Fatalf("unexpected refresh status: %#v", status)
	}

	data, err := os.ReadFile(filepath.Join(store.dir, peer.Name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatal("secret leaked into remote observation")
	}
}

func TestRegistryRejectsTransportHeaderOverride(t *testing.T) {
	peer := testPeer("https://devbox.example.test/v1/federation/snapshot")
	peer.Credentials = &CredentialRef{Type: "env_headers", Headers: map[string]string{"Host": "SPECVIEW_HOST_OVERRIDE"}}
	if err := ValidatePeer(peer); err == nil {
		t.Fatal("Host credential header must be rejected")
	}
}

func testPeer(url string) Peer {
	return Peer{Name: "devbox", URL: url, ExpectedHostID: testHostID, StaleAfterSeconds: DefaultStaleAfterSeconds}
}

func testSnapshot(observedAt time.Time) federation.HostSnapshot {
	return federation.HostSnapshot{
		SchemaVersion: federation.SnapshotSchemaVersion,
		HostID:        testHostID,
		Hostname:      "devbox-01",
		ObservedAt:    observedAt,
		Instances:     []federation.RepositoryInstance{},
	}
}

func snapshotPointer(snapshot federation.HostSnapshot) *federation.HostSnapshot {
	copySnapshot := snapshot
	return &copySnapshot
}
