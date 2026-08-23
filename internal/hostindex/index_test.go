package hostindex

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/hoststate"
)

func TestIndexSyncSearchAndRebuild(t *testing.T) {
	index, err := Open(filepath.Join(t.TempDir(), "index.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()

	now := time.Date(2026, 8, 22, 1, 0, 0, 0, time.UTC)
	repositories := []hoststate.Repository{
		{
			ID:          "repo-wms",
			Name:        "wms",
			Root:        "/work/spotwo/wms",
			FirstSeenAt: now.Add(-time.Hour),
			LastSeenAt:  now,
			Convention:  config.Convention{},
			Sessions: []hoststate.Session{{
				ID:           "session-codex",
				IdentityKind: hoststate.SessionIdentityLogical,
				Adapter:      "codex",
				Agent:        "Codex",
				ProcessIDs:   []int{10},
				StartedAt:    now.Add(-time.Minute),
				LastSeenAt:   now,
				Active:       true,
			}},
		},
		{
			ID:          "repo-api",
			Name:        "candidate-api",
			Root:        "/work/hiring/candidate-api",
			FirstSeenAt: now.Add(-2 * time.Hour),
			LastSeenAt:  now.Add(-time.Hour),
			Convention: config.Convention{
				Adapter:    config.AdapterGitHubSpecKit,
				Label:      "GitHub Spec Kit",
				Path:       "specs",
				Recognized: true,
				Supported:  true,
			},
			Sessions: []hoststate.Session{{
				ID:           "session-claude",
				IdentityKind: hoststate.SessionIdentityLogical,
				Adapter:      "claude",
				Agent:        "Claude",
				ProcessIDs:   []int{20},
				StartedAt:    now.Add(-90 * time.Minute),
				LastSeenAt:   now.Add(-time.Hour),
				Active:       false,
			}},
		},
	}

	if err := index.Sync(context.Background(), repositories); err != nil {
		t.Fatal(err)
	}

	assertSearch := func(query, want string) {
		t.Helper()
		ids, err := index.SearchRepositoryIDs(context.Background(), query, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 || ids[0] != want {
			t.Fatalf("search %q = %#v, want %q", query, ids, want)
		}
	}
	assertSearch("wms", "repo-wms")
	assertSearch("spotwo", "repo-wms")
	assertSearch("codex", "repo-wms")
	assertSearch("claude", "repo-api")
	assertSearch("spec kit", "repo-api")

	if err := index.Sync(context.Background(), repositories[1:]); err != nil {
		t.Fatal(err)
	}
	ids, err := index.SearchRepositoryIDs(context.Background(), "wms", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("rebuild retained removed repository: %#v", ids)
	}
}

func TestDefaultPathLivesBesideCatalog(t *testing.T) {
	catalog := filepath.Join(t.TempDir(), "Specview", "catalog.json")
	if got, want := DefaultPath(catalog), filepath.Join(filepath.Dir(catalog), "index.sqlite"); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}
