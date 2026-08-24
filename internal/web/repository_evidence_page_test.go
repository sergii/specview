package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

func TestRepositoryEvidencePageRendersNativeEvidenceWithoutSourceControl(t *testing.T) {
	root := t.TempDir()
	if err := writeAcceptanceOverviewFixture(root); err != nil {
		t.Fatal(err)
	}
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 15, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{ID: "evidence-session", Adapter: "codex", Agent: "Codex", RepositoryRoot: root, WorktreeRoot: root, CWD: root, ProcessIDs: []int{81}, StartedAt: now}}, now); err != nil {
		t.Fatal(err)
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repositories))
	}

	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.repositoryEvidencePage(recorder, httptest.NewRequest(http.MethodGet, "/project/evidence?id="+repositories[0].ID, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-specview="evidence-overview"`,
		`data-evidence-count="1"`,
		`data-passed="1"`,
		`data-evidence-id="H33-unit-tests"`,
		`data-evidence-result="passed"`,
		`H33 Repository Acceptance`,
		`unit-tests`,
		`fixture`,
		`git:abc123`,
		`2026-08-24T14:00:00Z`,
		`href="/project/spec?id=` + repositories[0].ID + `&path=H33.md"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("evidence overview missing %q; body=%s", want, body)
		}
	}
}

func TestRepositoryEvidencePageRejectsUnknownRepository(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	server.repositoryEvidencePage(recorder, httptest.NewRequest(http.MethodGet, "/project/evidence?id=missing", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", recorder.Code, recorder.Body.String())
	}
}
