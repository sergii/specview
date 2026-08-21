package specs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSpecWithoutMetadataDefaultsToNew(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "H01.md")
	if err := os.WriteFile(path, []byte("# First spec\n\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	item, err := parseFile(path, "H01.md")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != StatusNew {
		t.Fatalf("expected new, got %q", item.Status)
	}
	if item.Title != "First spec" {
		t.Fatalf("unexpected title %q", item.Title)
	}
	if item.ID != "H01" {
		t.Fatalf("unexpected id %q", item.ID)
	}
	if item.Kind != ArtifactSpec {
		t.Fatalf("unexpected kind %q", item.Kind)
	}
}

func TestSpecReadsStatus(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "H02.md")
	data := []byte("---\nspecview:\n  status: in_progress\n---\n# Outbox\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	item, err := parseFile(path, "H02.md")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != StatusInProgress || item.Error != "" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestUnknownStatusIsVisibleAsMetadataError(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "H03.md")
	data := []byte("---\nspecview:\n  status: shipped\n---\n# NATS\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	item, err := parseFile(path, "H03.md")
	if err != nil {
		t.Fatal(err)
	}
	if item.Error == "" {
		t.Fatal("expected metadata validation error")
	}
}

func TestStoreScansNestedSpecs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "group"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "group", "H04.md"), []byte("# Nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(root, "*.md")
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	items := store.All()
	if len(items) != 1 || items[0].Path != "group/H04.md" {
		t.Fatalf("unexpected items: %#v", items)
	}
	if store.AdapterName() != SpecviewAdapterName {
		t.Fatalf("unexpected adapter %q", store.AdapterName())
	}
}

func TestNewAdapterDefaultsToSpecview(t *testing.T) {
	adapter, err := NewAdapter("", t.TempDir(), "*.md")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Name() != SpecviewAdapterName {
		t.Fatalf("adapter name = %q, want %q", adapter.Name(), SpecviewAdapterName)
	}
}

func TestNewAdapterRejectsUnsupportedAdapter(t *testing.T) {
	if _, err := NewAdapter("unknown", t.TempDir(), "*.md"); err == nil {
		t.Fatal("expected unsupported adapter error")
	}
}

type stubAdapter struct {
	items []Spec
}

func (a stubAdapter) Name() string {
	return "stub"
}

func (a stubAdapter) Scan() ([]Spec, error) {
	return a.items, nil
}

func (a stubAdapter) WatchRoots() []string {
	return nil
}

func TestStoreUsesAdapterBoundary(t *testing.T) {
	store := NewStoreWithAdapter(stubAdapter{items: []Spec{{ID: "A1", Kind: ArtifactRFC, Title: "Decision input"}}})
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	items := store.All()
	if len(items) != 1 || items[0].Kind != ArtifactRFC {
		t.Fatalf("unexpected normalized artifacts: %#v", items)
	}
	if store.AdapterName() != "stub" {
		t.Fatalf("unexpected adapter %q", store.AdapterName())
	}
}
