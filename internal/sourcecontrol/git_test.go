package sourcecontrol

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInspectGitProjectsWorktreeSyncAndDirtyState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "init")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "add", "README.md")
	runTestGit(t, root, "-c", "user.name=Specview Test", "-c", "user.email=specview@example.invalid", "commit", "-m", "initial")
	runTestGit(t, root, "branch", "upstream")
	runTestGit(t, root, "branch", "--set-upstream-to=upstream")
	runTestGit(t, root, "remote", "add", "origin", "git@github.com:spotwo/wms.git")

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "add", "README.md")
	runTestGit(t, root, "-c", "user.name=Specview Test", "-c", "user.email=specview@example.invalid", "commit", "-m", "ahead")
	if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	context, err := InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	if context.Remote != "git@github.com:spotwo/wms.git" {
		t.Fatalf("remote = %q", context.Remote)
	}
	if len(context.Worktrees) != 1 {
		t.Fatalf("worktrees = %#v", context.Worktrees)
	}
	worktree := context.Worktrees[0]
	if worktree.Upstream != "upstream" || worktree.Ahead != 1 || worktree.Behind != 0 {
		t.Fatalf("sync state = %#v", worktree)
	}
	if worktree.DirtyCount != 1 {
		t.Fatalf("dirty count = %d", worktree.DirtyCount)
	}
	if worktree.LastCommit != "ahead" {
		t.Fatalf("last commit = %q", worktree.LastCommit)
	}
	if worktree.SyncLabel() != "upstream · ↑1" {
		t.Fatalf("sync label = %q", worktree.SyncLabel())
	}
}

func TestParseWorktrees(t *testing.T) {
	worktrees := ParseWorktrees(`worktree /work/repo
HEAD 0123456789abcdef
branch refs/heads/main

worktree /work/repo-feature
HEAD fedcba9876543210
branch refs/heads/feature

worktree /work/repo-detached
HEAD aabbccddeeff0011
detached
`)
	if len(worktrees) != 3 {
		t.Fatalf("worktrees = %#v", worktrees)
	}
	if worktrees[0].Branch != "main" || worktrees[1].Branch != "feature" || !worktrees[2].Detached {
		t.Fatalf("worktrees = %#v", worktrees)
	}
}

func runTestGit(t *testing.T, root string, args ...string) {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
