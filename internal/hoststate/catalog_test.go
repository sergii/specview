package hoststate

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/config"
)

func TestCatalogPersistsRepositorySessionsAndHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	detector := func(string) (config.Convention, error) {
		return config.Convention{
			Adapter:    config.AdapterGitHubSpecKit,
			Label:      "GitHub Spec Kit",
			Path:       "specs",
			Recognized: true,
			Supported:  true,
		}, nil
	}
	catalog, err := openCatalog(path, detector)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.Local)
	_, err = catalog.Observe([]Observation{{
		Agent:          "Codex",
		PID:            123,
		RepositoryRoot: "/work/specview",
	}}, now)
	if err != nil {
		t.Fatal(err)
	}

	repositories := catalog.Repositories()
	if len(repositories) != 1 {
		t.Fatalf("repositories = %#v", repositories)
	}
	repo := repositories[0]
	if !repo.Active() || repo.ActiveAgentLabel() != "Codex" {
		t.Fatalf("unexpected active repository: %#v", repo)
	}
	if repo.SpecificationLabel() != "GitHub Spec Kit" {
		t.Fatalf("unexpected convention label %q", repo.SpecificationLabel())
	}

	later := now.Add(10 * time.Minute)
	if _, err := catalog.Observe(nil, later); err != nil {
		t.Fatal(err)
	}
	repo = catalog.Repositories()[0]
	if repo.Active() || repo.Sessions[0].EndedAt == nil {
		t.Fatalf("session should be historical: %#v", repo.Sessions)
	}

	reloaded, err := openCatalog(path, detector)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Repositories(); len(got) != 1 || got[0].Sessions[0].EndedAt == nil {
		t.Fatalf("catalog did not persist history: %#v", got)
	}
}

func TestLooksLikeCodex(t *testing.T) {
	for _, command := range []string{
		"codex",
		"/usr/local/bin/codex --full-auto",
		"node /opt/node_modules/@openai/codex/bin/codex.js",
	} {
		if !looksLikeCodex(command) {
			t.Fatalf("expected Codex command: %q", command)
		}
	}
	if looksLikeCodex("node server.js") {
		t.Fatal("ordinary node process must not match Codex")
	}
}
