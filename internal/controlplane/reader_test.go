package controlplane

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

type stubExecutionSource struct {
	sessions []hoststate.ExecutionSession
	err      error
}

func (s stubExecutionSource) Sessions() ([]hoststate.ExecutionSession, error) {
	return append([]hoststate.ExecutionSession(nil), s.sessions...), s.err
}

type stubSourceControl struct {
	contexts map[string]sourcecontrol.RepositoryContext
	err      error
}

func (s stubSourceControl) Inspect(_ context.Context, root string) (sourcecontrol.RepositoryContext, error) {
	if s.err != nil {
		return sourcecontrol.RepositoryContext{}, s.err
	}
	return s.contexts[filepath.Clean(root)], nil
}

func TestListRepositoriesCombinesHistoryWithLiveExecution(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sergii", "specview")
	statePath := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	if _, err := catalog.Observe([]hoststate.Observation{{
		Agent:          "Codex",
		PID:            100,
		RepositoryRoot: root,
	}}, observedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe(nil, observedAt.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	liveRoot := filepath.Join(t.TempDir(), "teplotec", "infra")
	reader := NewReader(statePath, stubExecutionSource{sessions: []hoststate.ExecutionSession{{
		Adapter:        "claude",
		ID:             "execution-live",
		Agent:          "Claude",
		CWD:            liveRoot,
		RepositoryRoot: liveRoot,
		WorktreeRoot:   liveRoot,
		ProcessIDs:     []int{200},
	}}}, stubSourceControl{})

	result, err := reader.ListRepositories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != SchemaVersion || result.Host == "" {
		t.Fatalf("unexpected result metadata: %#v", result)
	}
	if len(result.Repositories) != 2 {
		t.Fatalf("repositories = %#v, want history plus live transient repository", result.Repositories)
	}
	if !result.Repositories[0].Active || result.Repositories[0].Name != "teplotec/infra" {
		t.Fatalf("live repository must sort first: %#v", result.Repositories[0])
	}
	if len(result.Repositories[0].Agents) != 1 || result.Repositories[0].Agents[0] != "Claude" {
		t.Fatalf("unexpected live agents: %#v", result.Repositories[0].Agents)
	}
	if result.Repositories[1].Active {
		t.Fatalf("historical repository must not inherit stale catalog activity: %#v", result.Repositories[1])
	}
}

func TestGetRepositoryProjectsGitAndForgeWithoutOwningThem(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sergii", "specview")
	statePath := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 100, RepositoryRoot: root}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	repository := catalog.Repositories()[0]

	reader := NewReader(
		statePath,
		stubExecutionSource{sessions: []hoststate.ExecutionSession{{
			ID:             "execution-1",
			Adapter:        "codex",
			Agent:          "Codex",
			RepositoryRoot: root,
			WorktreeRoot:   root,
			CWD:            root,
		}}},
		stubSourceControl{contexts: map[string]sourcecontrol.RepositoryContext{
			filepath.Clean(root): {
				Git: sourcecontrol.GitContext{
					Remote: "https://github.com/sergii/specview.git",
					Worktrees: []sourcecontrol.Worktree{{
						Path:       root,
						Branch:     "feat/mcp-server",
						Head:       "abc123",
						DirtyCount: 2,
					}},
				},
				Provider: sourcecontrol.ProviderContext{
					Name:       "GitHub",
					Matched:    true,
					Available:  true,
					Repository: "sergii/specview",
					WebURL:     "https://github.com/sergii/specview",
					PullRequests: []sourcecontrol.PullRequest{{
						Number: 4,
						Title:  "H18 MCP",
						State:  "OPEN",
						Head:   "feat/mcp-server",
						Base:   "main",
						Checks: sourcecontrol.CheckSummary{Total: 2, Passed: 1, Pending: 1},
					}},
				},
			},
		}},
	)

	result, err := reader.GetRepository(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Repository.Active || len(result.Repository.Agents) != 1 || result.Repository.Agents[0] != "Codex" {
		t.Fatalf("unexpected repository execution projection: %#v", result.Repository)
	}
	if result.Repository.Git == nil || len(result.Repository.Git.Worktrees) != 1 || result.Repository.Git.Worktrees[0].DirtyCount != 2 {
		t.Fatalf("unexpected Git projection: %#v", result.Repository.Git)
	}
	if result.Repository.Forge == nil || result.Repository.Forge.Provider != "GitHub" || len(result.Repository.Forge.PullRequests) != 1 {
		t.Fatalf("unexpected forge projection: %#v", result.Repository.Forge)
	}
}

func TestListActiveSessionsUsesLiveExecutionIdentity(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "sergii", "specview")
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 100, RepositoryRoot: root}}, time.Now()); err != nil {
		t.Fatal(err)
	}

	startedAt := time.Date(2026, 8, 23, 12, 30, 0, 0, time.UTC)
	reader := NewReader(statePath, stubExecutionSource{sessions: []hoststate.ExecutionSession{{
		ID:             "execution-1",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   filepath.Join(root, ".worktrees", "h18"),
		CWD:            filepath.Join(root, ".worktrees", "h18"),
		ProcessIDs:     []int{10, 11},
		StartedAt:      startedAt,
	}}}, stubSourceControl{})

	result, err := reader.ListActiveSessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("sessions = %#v", result.Sessions)
	}
	session := result.Sessions[0]
	if session.RepositoryID != hoststate.RepositoryIDForRoot(root) || session.StartedAt != startedAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected session contract: %#v", session)
	}
}

func TestListWorktreesReturnsDegradableSourceWarning(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "local", "scratch")
	if _, err := catalog.Observe([]hoststate.Observation{{Agent: "Codex", PID: 100, RepositoryRoot: root}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	repository := catalog.Repositories()[0]

	reader := NewReader(statePath, stubExecutionSource{}, stubSourceControl{err: context.DeadlineExceeded})
	result, err := reader.ListWorktrees(context.Background(), repository.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Worktrees) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("source failure should degrade: %#v", result)
	}
}
