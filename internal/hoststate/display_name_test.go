package hoststate

import (
	"path/filepath"
	"testing"
)

func TestRepositoryDisplayNameUsesParentAndRepository(t *testing.T) {
	root := filepath.Join(t.TempDir(), "spotwo", "wms")
	if got, want := repositoryDisplayName(root), "spotwo/wms"; got != want {
		t.Fatalf("display name = %q, want %q", got, want)
	}
}

func TestRepositoryDisplayNamePreservesCase(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Varkada", "Platform")
	if got, want := repositoryDisplayName(root), "Varkada/Platform"; got != want {
		t.Fatalf("display name = %q, want %q", got, want)
	}
}
