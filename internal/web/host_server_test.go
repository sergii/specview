package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

func TestHostDashboardAndProjectProjection(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "candidate-api")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".specify"), 0o755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", repoRoot, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	featureRoot := filepath.Join(repoRoot, "specs", "001-feedback")
	if err := os.MkdirAll(featureRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featureRoot, "spec.md"), []byte("# Candidate Feedback\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := catalog.Observe([]hoststate.Observation{{
		Agent:          "Codex",
		PID:            321,
		RepositoryRoot: repoRoot,
	}}, now); err != nil {
		t.Fatal(err)
	}

	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)

	response := httptest.NewRecorder()
	server.index(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("host status = %d", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{"candidate-api", "Codex", "GitHub Spec Kit", "Today"} {
		if !strings.Contains(body, want) {
			t.Fatalf("host dashboard missing %q", want)
		}
	}

	repo := catalog.Repositories()[0]
	response = httptest.NewRecorder()
	server.project(response, httptest.NewRequest(http.MethodGet, "/project?id="+repo.ID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("project status = %d body=%s", response.Code, response.Body.String())
	}
	body = response.Body.String()
	for _, want := range []string{"Candidate Feedback", "GitHub Spec Kit", "New", "Worktrees", "GitHub"} {
		if !strings.Contains(body, want) {
			t.Fatalf("project page missing %q", want)
		}
	}
}

func TestHostDashboardEmptyToday(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	response := httptest.NewRecorder()
	server.index(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(response.Body.String(), "You haven't run anything yet.") {
		t.Fatalf("empty state missing: %s", response.Body.String())
	}
}

type fakeRepositorySearcher struct {
	ids []string
}

func (f fakeRepositorySearcher) SearchRepositoryIDs(context.Context, string, int) ([]string, error) {
	return append([]string(nil), f.ids...), nil
}

func TestHostDashboardSearchUsesIndexIdentityAndLiveCatalogProjection(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for pid, root := range map[int]string{
		101: filepath.Join(t.TempDir(), "wms"),
		202: filepath.Join(t.TempDir(), "candidate-api"),
	} {
		if _, err := catalog.Observe([]hoststate.Observation{{
			Agent:          "Codex",
			PID:            pid,
			RepositoryRoot: root,
		}}, now); err != nil {
			t.Fatal(err)
		}
	}

	repositories := catalog.Repositories()
	var wanted hoststate.Repository
	for _, repository := range repositories {
		if repository.Name == "wms" {
			wanted = repository
			break
		}
	}
	if wanted.ID == "" {
		t.Fatal("wms repository missing from test catalog")
	}

	server := NewHostServerWithSources(
		catalog,
		NewHub(),
		"127.0.0.1",
		7331,
		nil,
		nil,
		fakeRepositorySearcher{ids: []string{wanted.ID}},
	)
	response := httptest.NewRecorder()
	server.index(response, httptest.NewRequest(http.MethodGet, "/?q=spotwo", nil))
	body := response.Body.String()
	if !strings.Contains(body, "Results") || !strings.Contains(body, "wms") {
		t.Fatalf("search result missing: %s", body)
	}
	if strings.Contains(body, "candidate-api") {
		t.Fatalf("search leaked unmatched repository: %s", body)
	}
}
