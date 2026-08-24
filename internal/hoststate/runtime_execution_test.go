package hoststate

import (
	"path/filepath"
	"testing"
	"time"
)

type logicalRuntimeSource struct {
	sessions     []ExecutionSession
	scanCalls    int
	sessionCalls int
}

func (s *logicalRuntimeSource) Scan() ([]Observation, error) {
	s.scanCalls++
	return []Observation{{Agent: "Codex", PID: 9999, RepositoryRoot: "/wrong/process/path"}}, nil
}

func (s *logicalRuntimeSource) Sessions() ([]ExecutionSession, error) {
	s.sessionCalls++
	return append([]ExecutionSession(nil), s.sessions...), nil
}

func TestRuntimePrefersLogicalExecutionSourceOverProcessScanner(t *testing.T) {
	catalog, err := openCatalog(filepath.Join(t.TempDir(), "catalog.json"), logicalTestDetector)
	if err != nil {
		t.Fatal(err)
	}
	source := &logicalRuntimeSource{sessions: []ExecutionSession{{
		Adapter:        "codex",
		ID:             "execution-runtime",
		Agent:          "Codex",
		CWD:            "/work/specview",
		RepositoryRoot: "/work/specview",
		WorktreeRoot:   "/work/specview",
		ProcessIDs:     []int{30, 10, 20},
	}}}

	runtime := NewRuntime(catalog, source, time.Hour, nil)
	count, err := runtime.Refresh()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || source.sessionCalls != 1 || source.scanCalls != 0 {
		t.Fatalf("runtime path count=%d sessionCalls=%d scanCalls=%d", count, source.sessionCalls, source.scanCalls)
	}

	repositories := catalog.Repositories()
	if len(repositories) != 1 || len(repositories[0].Sessions) != 1 {
		t.Fatalf("runtime did not persist one logical session: %#v", repositories)
	}
	session := repositories[0].Sessions[0]
	if session.ID != "execution-runtime" || session.IdentityKind != SessionIdentityLogical {
		t.Fatalf("runtime persisted process-shaped identity: %#v", session)
	}
	if !equalIntSlices(session.ProcessIDs, []int{10, 20, 30}) {
		t.Fatalf("runtime process diagnostics = %#v", session.ProcessIDs)
	}
}
