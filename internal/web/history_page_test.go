package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

func TestHistoryPageShowsActiveAndEndedSessions(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	now := time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC)
	ended := hoststate.ExecutionSession{ID: "ended", Adapter: "claude-code", Agent: "Claude", RepositoryRoot: root, WorktreeRoot: root, CWD: root, ProcessIDs: []int{31}, StartedAt: now.Add(-time.Minute)}
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{ended}, now); err != nil {
		t.Fatal(err)
	}
	liveAt := now.Add(5 * time.Minute)
	live := hoststate.ExecutionSession{ID: "live", Adapter: "codex", Agent: "Codex", RepositoryRoot: root, WorktreeRoot: root, CWD: root, ProcessIDs: []int{41, 42}, StartedAt: liveAt.Add(-time.Minute)}
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{live}, liveAt); err != nil {
		t.Fatal(err)
	}

	repos := catalog.Repositories()
	if len(repos) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repos))
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.historyPage(recorder, httptest.NewRequest(http.MethodGet, "/history", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{`data-session-id="live"`, `data-session-active="true"`, `data-session-id="ended"`, `data-session-active="false"`, `data-identity-kind="logical"`, `href="/project?id=` + repos[0].ID + `"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("history page missing %q", want)
		}
	}
}
