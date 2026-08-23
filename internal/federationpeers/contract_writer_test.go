package federationpeers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPeerRegistryWriterMatchesV1Fixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "contracts", "peers", "v1.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := decodeRegistry(fixture)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "federation-peers.json")
	registry, err := OpenRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := len(persisted.Peers) - 1; i >= 0; i-- {
		if err := registry.Add(persisted.Peers[i]); err != nil {
			t.Fatal(err)
		}
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, fixture) {
		t.Fatalf("peer registry writer changed v1 contract\nactual:\n%s\nwant:\n%s", actual, fixture)
	}
}

func TestRemoteObservationWriterMatchesV1Fixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "contracts", "remote-observation", "v1.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	observation, err := decodeObservation(fixture)
	if err != nil {
		t.Fatal(err)
	}

	store := NewObservationStore(t.TempDir())
	if err := store.save(observation); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(store.dir, observation.Peer+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, fixture) {
		t.Fatalf("remote observation writer changed v1 contract\nactual:\n%s\nwant:\n%s", actual, fixture)
	}
}
