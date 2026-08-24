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

func TestHistorySessionPageRendersExactLogicalSession(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "session-34",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root + "/specs",
		ProcessIDs:     []int{3402, 3401},
		StartedAt:      now.Add(-5 * time.Minute),
	}}, now); err != nil {
		t.Fatal(err)
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repositories))
	}

	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	query := url.Values{"repository": {repositories[0].ID}, "session": {"session-34"}}
	recorder := httptest.NewRecorder()
	server.historySessionPage(recorder, httptest.NewRequest(http.MethodGet, "/history/session?"+query.Encode(), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-session-id="session-34"`,
		`data-session-active="true"`,
		`data-identity-kind="logical"`,
		`Codex`,
		`codex`,
		`3401, 3402`,
		`2026-08-24T11:55:00Z`,
		`2026-08-24T12:00:00Z`,
		`href="/project?id=` + repositories[0].ID + `"`,
		`href="/history?repository=` + repositories[0].ID + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("session detail missing %q; body=%s", want, body)
		}
	}
}

func TestHistorySessionPageRejectsWrongRepositorySessionPair(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID: "shared", Adapter: "codex", Agent: "Codex", RepositoryRoot: root, WorktreeRoot: root, StartedAt: now,
	}}, now); err != nil {
		t.Fatal(err)
	}

	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.historySessionPage(recorder, httptest.NewRequest(http.MethodGet, "/history/session?repository=missing&session=shared", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}
