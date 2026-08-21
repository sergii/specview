package hoststate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeExecutionAdapter struct {
	name     string
	sessions []ExecutionSession
	err      error
}

func (f fakeExecutionAdapter) Name() string { return f.name }

func (f fakeExecutionAdapter) Sessions() ([]ExecutionSession, error) {
	return append([]ExecutionSession(nil), f.sessions...), f.err
}

func (f fakeExecutionAdapter) Diagnostics() ([]ScanDiagnostic, error) {
	return nil, f.err
}

func TestExecutionRegistryAggregatesHealthyAdapters(t *testing.T) {
	registry := NewExecutionRegistry(
		fakeExecutionAdapter{name: "codex", sessions: []ExecutionSession{{
			Adapter:        "codex",
			ID:             "execution-codex",
			Agent:          "Codex",
			RepositoryRoot: "/repos/a",
			CWD:            "/repos/a",
			ProcessIDs:     []int{10, 11},
		}}},
		fakeExecutionAdapter{name: "other", sessions: []ExecutionSession{{
			Adapter:        "other",
			ID:             "execution-other",
			Agent:          "Other",
			RepositoryRoot: "/repos/b",
			CWD:            "/repos/b",
			ProcessIDs:     []int{20},
		}}},
	)

	sessions, err := registry.Sessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("sessions = %#v", sessions)
	}

	observations, err := registry.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 3 {
		t.Fatalf("observations = %#v", observations)
	}
}

func TestExecutionRegistryPreservesHealthySessionsWhenAdapterFails(t *testing.T) {
	registry := NewExecutionRegistry(
		fakeExecutionAdapter{name: "broken", err: errors.New("boom")},
		fakeExecutionAdapter{name: "healthy", sessions: []ExecutionSession{{
			Adapter:        "healthy",
			ID:             "execution-healthy",
			Agent:          "Healthy",
			RepositoryRoot: "/repos/healthy",
			CWD:            "/repos/healthy",
			ProcessIDs:     []int{42},
		}}},
	)

	sessions, err := registry.Sessions()
	if err != nil {
		t.Fatalf("healthy sessions should survive another adapter failure: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Adapter != "healthy" {
		t.Fatalf("sessions = %#v", sessions)
	}
}

func TestExecutionRegistryReturnsErrorWhenAllAdaptersFail(t *testing.T) {
	registry := NewExecutionRegistry(fakeExecutionAdapter{name: "broken", err: errors.New("boom")})
	if _, err := registry.Sessions(); err == nil {
		t.Fatal("expected registry error when no adapter can produce sessions")
	}
}

func TestSessionsFromDiagnosticsGroupsProcessesByRepositoryAndCWD(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, root, "init")

	diagnostics := []ScanDiagnostic{
		{PID: 101, Matched: true, CWD: root, RepositoryRoot: root, Stage: "ok"},
		{PID: 102, Matched: true, CWD: root, RepositoryRoot: root, Stage: "ok"},
	}
	sessions := sessionsFromDiagnostics("codex", "Codex", diagnostics)
	if len(sessions) != 1 {
		t.Fatalf("sessions = %#v", sessions)
	}
	session := sessions[0]
	if session.Adapter != "codex" || session.Agent != "Codex" {
		t.Fatalf("session = %#v", session)
	}
	if len(session.ProcessIDs) != 2 || session.ProcessIDs[0] != 101 || session.ProcessIDs[1] != 102 {
		t.Fatalf("process ids = %#v", session.ProcessIDs)
	}
	if session.ID == "" {
		t.Fatal("logical session ID must be populated")
	}
	if session.WorktreeRoot != filepath.Clean(root) {
		t.Fatalf("worktree root = %q", session.WorktreeRoot)
	}
}
