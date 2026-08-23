package hoststate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseWorktrees(t *testing.T) {
	worktrees := parseWorktrees(`worktree /work/specview
HEAD 0123456789abcdef
branch refs/heads/main

worktree /work/specview-feature
HEAD fedcba9876543210
branch refs/heads/feat/h13

worktree /work/specview-detached
HEAD aabbccddeeff0011
detached
`)
	if len(worktrees) != 3 {
		t.Fatalf("worktrees = %#v", worktrees)
	}
	if worktrees[0].Path != "/work/specview" || worktrees[0].Branch != "main" {
		t.Fatalf("unexpected primary worktree: %#v", worktrees[0])
	}
	if worktrees[1].Branch != "feat/h13" {
		t.Fatalf("unexpected linked branch: %#v", worktrees[1])
	}
	if !worktrees[2].Detached || worktrees[2].BranchLabel() != "detached" {
		t.Fatalf("unexpected detached worktree: %#v", worktrees[2])
	}
}

func TestInspectGitRepositoryIncludesRemoteAndDirtyState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wms")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "init")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# WMS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "add", "README.md")
	runTestGit(t, root, "-c", "user.name=Specview Test", "-c", "user.email=specview@example.invalid", "commit", "-m", "initial")
	runTestGit(t, root, "remote", "add", "origin", "git@github.com:spotwo/wms.git")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# WMS\nchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	view, err := inspectGitRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if view.Remote != "git@github.com:spotwo/wms.git" {
		t.Fatalf("remote = %q", view.Remote)
	}
	if len(view.Worktrees) != 1 {
		t.Fatalf("worktrees = %#v", view.Worktrees)
	}
	if view.Worktrees[0].DirtyCount != 1 {
		t.Fatalf("dirty count = %d", view.Worktrees[0].DirtyCount)
	}
	if view.Worktrees[0].ChangeLabel() != "1 change" {
		t.Fatalf("change label = %q", view.Worktrees[0].ChangeLabel())
	}
}

func TestRepositoryExecutionViewUsesInjectedExecutionSource(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "init")

	started := time.Now().Add(-time.Minute)
	repo := Repository{
		Root: root,
		Sessions: []Session{{
			Agent:     "FutureAgent",
			PID:       777,
			StartedAt: started,
			Active:    true,
		}},
	}
	registry := NewExecutionRegistry(fakeExecutionAdapter{
		name: "future",
		sessions: []ExecutionSession{{
			Adapter:        "future",
			ID:             "execution-future",
			Agent:          "FutureAgent",
			CWD:            root,
			RepositoryRoot: root,
			WorktreeRoot:   root,
			ProcessIDs:     []int{777},
		}},
	})

	view := repo.ExecutionView(registry)
	if view.Error != "" {
		t.Fatalf("execution view error = %q", view.Error)
	}
	if len(view.Sessions) != 1 || view.Sessions[0].Adapter != "future" {
		t.Fatalf("sessions = %#v", view.Sessions)
	}
	if !view.Sessions[0].StartedAt.Equal(started) {
		t.Fatalf("started at = %v, want %v", view.Sessions[0].StartedAt, started)
	}
	if len(view.Worktrees) != 1 || view.Worktrees[0].AgentLabel() != "FutureAgent" {
		t.Fatalf("worktrees = %#v", view.Worktrees)
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
