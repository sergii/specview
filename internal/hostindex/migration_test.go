package hostindex

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

func TestOpenRebuildsLegacyV1ProjectionAsSchemaV2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.sqlite")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`INSERT INTO meta(key, value) VALUES('schema_version', '1')`,
		`CREATE TABLE repositories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			root TEXT NOT NULL,
			first_seen_at INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			convention_label TEXT NOT NULL,
			detection_error TEXT NOT NULL
		)`,
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			repository_id TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
			agent TEXT NOT NULL,
			pid INTEGER NOT NULL,
			started_at INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			ended_at INTEGER,
			active INTEGER NOT NULL
		)`,
		`INSERT INTO repositories(id, name, root, first_seen_at, last_seen_at, convention_label, detection_error)
		 VALUES('legacy-repo', 'legacy', '/work/legacy', 1, 2, '', '')`,
		`INSERT INTO sessions(id, repository_id, agent, pid, started_at, last_seen_at, active)
		 VALUES('legacy-session', 'legacy-repo', 'Codex', 4242, 1, 2, 1)`,
	} {
		if _, err := legacy.Exec(statement); err != nil {
			_ = legacy.Close()
			t.Fatalf("prepare v1 projection: %v", err)
		}
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	index, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()

	var version string
	if err := index.db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != "2" {
		t.Fatalf("schema version = %q, want 2", version)
	}

	columns := make(map[string]bool)
	rows, err := index.db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if columns["pid"] || !columns["identity_kind"] || !columns["process_ids"] || !columns["adapter"] {
		t.Fatalf("unexpected v2 sessions columns: %#v", columns)
	}

	var repositoryCount, sessionCount int
	if err := index.db.QueryRow(`SELECT count(*) FROM repositories`).Scan(&repositoryCount); err != nil {
		t.Fatal(err)
	}
	if err := index.db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessionCount); err != nil {
		t.Fatal(err)
	}
	if repositoryCount != 0 || sessionCount != 0 {
		t.Fatalf("rebuild retained derived v1 rows: repositories=%d sessions=%d", repositoryCount, sessionCount)
	}

	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	repositories := []hoststate.Repository{{
		ID:          "repo-logical",
		Name:        "logical",
		Root:        "/work/logical",
		FirstSeenAt: now,
		LastSeenAt:  now,
		Sessions: []hoststate.Session{{
			ID:           "execution-logical",
			IdentityKind: hoststate.SessionIdentityLogical,
			Adapter:      "codex",
			Agent:        "Codex",
			ProcessIDs:   []int{10, 20},
			StartedAt:    now,
			LastSeenAt:   now,
			Active:       true,
		}},
	}}
	if err := index.Sync(context.Background(), repositories); err != nil {
		t.Fatal(err)
	}
	ids, err := index.SearchRepositoryIDs(context.Background(), "codex", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "repo-logical" {
		t.Fatalf("rebuilt v2 projection search = %#v", ids)
	}
}
