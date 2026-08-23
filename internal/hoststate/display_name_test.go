package hoststate

import (
	"path/filepath"
	"testing"

	"github.com/sergii/specview/internal/sourcecontrol"
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

func TestExecutionSessionDisplayCWDUsesCompactPath(t *testing.T) {
	cwd := filepath.Join(t.TempDir(), "spotwo", "wms")
	session := ExecutionSession{CWD: cwd}
	if got, want := session.DisplayCWD(), "spotwo/wms"; got != want {
		t.Fatalf("display cwd = %q, want %q", got, want)
	}
}

func TestWorktreeDisplayPathUsesCompactPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spotwo", "project-login-from-main")
	worktree := Worktree{Worktree: sourcecontrol.Worktree{Path: path}}
	if got, want := worktree.DisplayPath(), "spotwo/project-login-from-main"; got != want {
		t.Fatalf("display worktree path = %q, want %q", got, want)
	}
}
