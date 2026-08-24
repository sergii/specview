package executionhistory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

func TestBuildMatchesV1Contract(t *testing.T) {
	ended := time.Date(2026, 8, 24, 7, 21, 0, 0, time.UTC)
	legacyEnded := time.Date(2026, 8, 23, 22, 6, 0, 0, time.UTC)
	repositories := []hoststate.Repository{
		{
			ID:   "repo-b",
			Name: "teplotec/website",
			Root: "/work/teplotec/website",
			Sessions: []hoststate.Session{
				{
					ID:           "legacy-7",
					IdentityKind: hoststate.SessionIdentityLegacyPID,
					Adapter:      "codex",
					Agent:        "Codex",
					ProcessIDs:   []int{7},
					StartedAt:    time.Date(2026, 8, 23, 22, 0, 0, 0, time.UTC),
					LastSeenAt:   time.Date(2026, 8, 23, 22, 5, 0, 0, time.UTC),
					EndedAt:      &legacyEnded,
					Active:       false,
				},
				{
					ID:           "session-ended",
					IdentityKind: hoststate.SessionIdentityLogical,
					Adapter:      "claude-code",
					Agent:        "Claude",
					ProcessIDs:   []int{42},
					CWD:          "/work/teplotec/website",
					StartedAt:    time.Date(2026, 8, 24, 7, 0, 0, 0, time.UTC),
					LastSeenAt:   time.Date(2026, 8, 24, 7, 20, 0, 0, time.UTC),
					EndedAt:      &ended,
					Active:       false,
				},
			},
		},
		{
			ID:   "repo-a",
			Name: "sergii/specview",
			Root: "/work/sergii/specview",
			Sessions: []hoststate.Session{{
				ID:           "session-live",
				IdentityKind: hoststate.SessionIdentityLogical,
				Adapter:      "codex",
				Agent:        "Codex",
				ProcessIDs:   []int{5, 2},
				CWD:          "/work/sergii/specview",
				WorktreeRoot: "/work/sergii/specview",
				StartedAt:    time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC),
				LastSeenAt:   time.Date(2026, 8, 24, 8, 30, 0, 0, time.UTC),
				Active:       true,
			}},
		},
	}

	got := Build("laptop.local", repositories)
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "contracts", "execution-history", "v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want Projection
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projection mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildSortsTiesDeterministically(t *testing.T) {
	now := time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC)
	projection := Build("host", []hoststate.Repository{
		{ID: "repo-b", Name: "b", Root: "/b", Sessions: []hoststate.Session{{ID: "session-z", LastSeenAt: now}}},
		{ID: "repo-a", Name: "a", Root: "/a", Sessions: []hoststate.Session{{ID: "session-b", LastSeenAt: now}, {ID: "session-a", LastSeenAt: now}}},
	})
	got := []string{projection.Entries[0].RepositoryID + "/" + projection.Entries[0].SessionID, projection.Entries[1].RepositoryID + "/" + projection.Entries[1].SessionID, projection.Entries[2].RepositoryID + "/" + projection.Entries[2].SessionID}
	want := []string{"repo-a/session-a", "repo-a/session-b", "repo-b/session-z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %#v, want %#v", got, want)
	}
}
