package federationruntime

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sergii/specview/internal/federationpeers"
)

type refresherStub struct {
	mu       sync.Mutex
	calls    []string
	failures map[string]error
}

func (s *refresherStub) Refresh(_ context.Context, peer federationpeers.Peer) (federationpeers.PeerStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, peer.Name)
	if err := s.failures[peer.Name]; err != nil {
		return federationpeers.PeerStatus{Peer: peer, Freshness: federationpeers.FreshnessUnreachable}, err
	}
	return federationpeers.PeerStatus{Peer: peer, Freshness: federationpeers.FreshnessFresh}, nil
}

func (s *refresherStub) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = nil
}

func (s *refresherStub) called() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

func TestPollerReopensRegistryEachCycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "federation-peers.json")
	registry, err := federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add(testPeer("alpha", "host:11111111-1111-4111-9111-111111111111")); err != nil {
		t.Fatal(err)
	}

	refresher := &refresherStub{failures: map[string]error{}}
	poller, err := NewPoller(path, refresher, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := refresher.called(); len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("initial refresh calls = %#v", got)
	}

	registry, err = federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add(testPeer("beta", "host:22222222-2222-4222-9222-222222222222")); err != nil {
		t.Fatal(err)
	}
	refresher.reset()
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := refresher.called(); len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("added peer was not observed without restart: %#v", got)
	}

	registry, err = federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Remove("alpha"); err != nil {
		t.Fatal(err)
	}
	refresher.reset()
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := refresher.called(); len(got) != 1 || got[0] != "beta" {
		t.Fatalf("removed peer leaked into runtime: %#v", got)
	}
}

func TestPollerIsolatesPeerFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "federation-peers.json")
	registry, err := federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add(testPeer("alpha", "host:11111111-1111-4111-9111-111111111111")); err != nil {
		t.Fatal(err)
	}
	if err := registry.Add(testPeer("beta", "host:22222222-2222-4222-9222-222222222222")); err != nil {
		t.Fatal(err)
	}

	refresher := &refresherStub{failures: map[string]error{"alpha": errors.New("offline")}}
	poller, err := NewPoller(path, refresher, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := poller.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Peers != 2 || summary.Failed != 1 || summary.Succeeded != 1 || len(summary.Results) != 2 {
		t.Fatalf("unexpected refresh summary: %#v", summary)
	}
	if got := refresher.called(); len(got) != 2 {
		t.Fatalf("one peer failure prevented other refreshes: %#v", got)
	}
}

func TestPollerBroadcastsPeerRemovalToEmptyRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "federation-peers.json")
	registry, err := federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add(testPeer("alpha", "host:11111111-1111-4111-9111-111111111111")); err != nil {
		t.Fatal(err)
	}

	changes := 0
	poller, err := NewPoller(path, &refresherStub{failures: map[string]error{}}, time.Second, func() { changes++ })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	registry, err = federationpeers.OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Remove("alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := poller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if changes != 2 {
		t.Fatalf("change callback count = %d, want 2", changes)
	}
}

func TestPollerRunStopsOnContextCancellation(t *testing.T) {
	poller, err := NewPoller(filepath.Join(t.TempDir(), "federation-peers.json"), &refresherStub{failures: map[string]error{}}, 5*time.Millisecond, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		poller.Run(ctx)
		close(done)
	}()
	time.Sleep(15 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("poller did not stop after context cancellation")
	}
}

func testPeer(name, hostID string) federationpeers.Peer {
	return federationpeers.Peer{
		Name:              name,
		URL:               "https://" + name + ".example.ts.net/v1/federation/snapshot",
		ExpectedHostID:    hostID,
		StaleAfterSeconds: 300,
	}
}
