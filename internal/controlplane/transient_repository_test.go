package controlplane

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func TestListedTransientRepositoryIsImmediatelyReadable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sergii", "specview")
	statePath := filepath.Join(t.TempDir(), "catalog.json")
	session := hoststate.ExecutionSession{
		ID:             "execution-live",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{101, 102},
	}
	reader := NewReader(
		statePath,
		stubExecutionSource{sessions: []hoststate.ExecutionSession{session}},
		stubSourceControl{contexts: map[string]sourcecontrol.RepositoryContext{
			filepath.Clean(root): {
				Git: sourcecontrol.GitContext{Worktrees: []sourcecontrol.Worktree{{
					Path:   root,
					Branch: "fix/v001-release-boundary",
					Head:   "abc123",
				}}},
			},
		}},
	)

	listed, err := reader.ListRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Repositories) != 1 {
		t.Fatalf("repositories = %#v, want one live transient repository", listed.Repositories)
	}
	repositoryID := listed.Repositories[0].ID
	if repositoryID != hoststate.RepositoryIDForRoot(root) {
		t.Fatalf("repository ID = %q, want %q", repositoryID, hoststate.RepositoryIDForRoot(root))
	}

	detail, err := reader.GetRepository(context.Background(), repositoryID)
	if err != nil {
		t.Fatalf("GetRepository(%q) after ListRepositories: %v", repositoryID, err)
	}
	if detail.Repository.Root != filepath.Clean(root) || !detail.Repository.Active {
		t.Fatalf("unexpected transient repository detail: %#v", detail.Repository)
	}
	if len(detail.Repository.Agents) != 1 || detail.Repository.Agents[0] != "Codex" {
		t.Fatalf("unexpected transient repository agents: %#v", detail.Repository.Agents)
	}

	worktrees, err := reader.ListWorktrees(context.Background(), repositoryID)
	if err != nil {
		t.Fatalf("ListWorktrees(%q) after ListRepositories: %v", repositoryID, err)
	}
	if len(worktrees.Worktrees) != 1 || worktrees.Worktrees[0].Path != filepath.Clean(root) {
		t.Fatalf("unexpected transient repository worktrees: %#v", worktrees.Worktrees)
	}
}
