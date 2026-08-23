package revision

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sergii/specview/internal/sourcecontrol"
)

func TestResolveGitCleanWorktree(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repos", "specview")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{
		Path: root,
		Head: "abcdef1234567890",
	}}}

	resolution := ResolveGit(root, git)
	if !resolution.Available {
		t.Fatalf("Available = false, reason = %q", resolution.Reason)
	}
	if resolution.Revision != "git:abcdef1234567890" {
		t.Fatalf("Revision = %q", resolution.Revision)
	}
	if resolution.WorktreePath != root {
		t.Fatalf("WorktreePath = %q, want %q", resolution.WorktreePath, root)
	}
}

func TestResolveGitProjectInsideWorktree(t *testing.T) {
	worktree := filepath.Join(string(filepath.Separator), "repos", "monorepo")
	project := filepath.Join(worktree, "services", "api")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: worktree, Head: "abc"}}}

	resolution := ResolveGit(project, git)
	if !resolution.Available || resolution.Revision != "git:abc" {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
}

func TestResolveGitChoosesDeepestContainingWorktree(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repos", "specview")
	linked := filepath.Join(root, "worktrees", "acceptance")
	project := filepath.Join(linked, "cmd")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{
		{Path: root, Head: "base"},
		{Path: linked, Head: "feature"},
	}}

	resolution := ResolveGit(project, git)
	if !resolution.Available || resolution.Revision != "git:feature" {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
	if resolution.WorktreePath != linked {
		t.Fatalf("WorktreePath = %q, want %q", resolution.WorktreePath, linked)
	}
}

func TestResolveGitCanonicalizesSymlinkedProjectRoot(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real", "specview")
	if err := os.MkdirAll(realRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	aliasRoot := filepath.Join(t.TempDir(), "specview")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: realRoot, Head: "abc"}}}
	resolution := ResolveGit(aliasRoot, git)
	if !resolution.Available || resolution.Revision != "git:abc" {
		t.Fatalf("unexpected resolution through symlinked root: %#v", resolution)
	}
	if resolution.WorktreePath != realRoot {
		t.Fatalf("WorktreePath = %q, want %q", resolution.WorktreePath, realRoot)
	}
}

func TestResolveGitDirtyWorktreeFailsClosed(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repos", "specview")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{
		Path:       root,
		Head:       "abcdef",
		DirtyCount: 2,
	}}}

	resolution := ResolveGit(root, git)
	if resolution.Available {
		t.Fatalf("Available = true, want false: %#v", resolution)
	}
	if resolution.Revision != "" || resolution.Reason != ReasonDirtyWorktree {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
}

func TestResolveGitMissingHeadFailsClosed(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repos", "specview")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: root}}}

	resolution := ResolveGit(root, git)
	if resolution.Available || resolution.Reason != ReasonHeadUnavailable {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
}

func TestResolveGitWithoutContainingWorktreeFailsClosed(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repos", "specview")
	other := filepath.Join(string(filepath.Separator), "repos", "other")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: other, Head: "abc"}}}

	resolution := ResolveGit(root, git)
	if resolution.Available || resolution.Reason != ReasonWorktreeNotFound {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
}

func TestResolveGitDoesNotTreatSiblingPrefixAsContainingWorktree(t *testing.T) {
	parent := filepath.Join(string(filepath.Separator), "repos", "app")
	sibling := filepath.Join(string(filepath.Separator), "repos", "app-two")
	git := sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{Path: parent, Head: "abc"}}}

	resolution := ResolveGit(sibling, git)
	if resolution.Available || resolution.Reason != ReasonWorktreeNotFound {
		t.Fatalf("unexpected resolution: %#v", resolution)
	}
}
