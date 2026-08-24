package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestHistoryPageScopesExactRepository(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	now := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{
		{ID: "first-session", Adapter: "codex", Agent: "Codex", RepositoryRoot: firstRoot, WorktreeRoot: firstRoot, CWD: firstRoot, ProcessIDs: []int{51}, StartedAt: now.Add(-2 * time.Minute)},
		{ID: "second-session", Adapter: "claude-code", Agent: "Claude", RepositoryRoot: secondRoot, WorktreeRoot: secondRoot, CWD: secondRoot, ProcessIDs: []int{61}, StartedAt: now.Add(-time.Minute)},
	}, now); err != nil {
		t.Fatal(err)
	}

	var first hoststate.Repository
	for _, repository := range catalog.Repositories() {
		if repository.Root == firstRoot {
			first = repository
			break
		}
	}
	if first.ID == "" {
		t.Fatal("first repository not found")
	}

	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	query := url.Values{"repository": {first.ID}}
	recorder := httptest.NewRecorder()
	server.historyPage(recorder, httptest.NewRequest(http.MethodGet, "/history?"+query.Encode(), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-history-repository="` + first.ID + `"`,
		`data-session-id="first-session"`,
		`href="/project?id=` + first.ID + `"`,
		`href="/history">All history</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("scoped history missing %q; body=%s", want, body)
		}
	}
	if strings.Contains(body, `data-session-id="second-session"`) {
		t.Fatalf("scoped history leaked another repository session: %s", body)
	}
}

func TestHistoryPageRejectsUnknownRepositoryScope(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.historyPage(recorder, httptest.NewRequest(http.MethodGet, "/history?repository=missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", recorder.Code, recorder.Body.String())
	}
}
