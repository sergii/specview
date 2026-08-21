package hostindex

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/sergii/specview/internal/hoststate"
	_ "modernc.org/sqlite"
)

const schemaVersion = 1

// RepositorySearcher is the read side consumed by the host UI.
type RepositorySearcher interface {
	SearchRepositoryIDs(context.Context, string, int) ([]string, error)
}

// Index is a rebuildable SQLite projection of host repository/session state.
// Filesystem, Git, execution adapters, and the host catalog remain authoritative.
type Index struct {
	db   *sql.DB
	path string

	mu          sync.Mutex
	fingerprint string
}

func DefaultPath(catalogPath string) string {
	if catalogPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(catalogPath), "index.sqlite")
}

func Open(path string) (*Index, error) {
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
	} else {
		path = ":memory:"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	index := &Index{db: db, path: path}
	if err := index.configure(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := index.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return index, nil
}

func (i *Index) Path() string { return i.path }

func (i *Index) Close() error { return i.db.Close() }

func (i *Index) configure(ctx context.Context) error {
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := i.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure SQLite host index: %w", err)
		}
	}
	return nil
}

func (i *Index) migrate(ctx context.Context) error {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS repositories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			root TEXT NOT NULL,
			first_seen_at INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			convention_label TEXT NOT NULL,
			detection_error TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			repository_id TEXT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
			agent TEXT NOT NULL,
			pid INTEGER NOT NULL,
			started_at INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			ended_at INTEGER,
			active INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS repositories_name_idx ON repositories(name)`,
		`CREATE INDEX IF NOT EXISTS repositories_root_idx ON repositories(root)`,
		`CREATE INDEX IF NOT EXISTS sessions_repository_idx ON sessions(repository_id)`,
		`CREATE INDEX IF NOT EXISTS sessions_agent_idx ON sessions(agent)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate SQLite host index: %w", err)
		}
	}

	var version string
	err = tx.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, `INSERT INTO meta(key, value) VALUES('schema_version', ?)`, fmt.Sprint(schemaVersion)); err != nil {
			return err
		}
	case err != nil:
		return err
	case version != fmt.Sprint(schemaVersion):
		return fmt.Errorf("unsupported SQLite host index schema version %s", version)
	}
	return tx.Commit()
}

// Sync replaces the derived projection when repository/session identity changes.
// Last-seen heartbeats alone do not rewrite SQLite every polling interval.
func (i *Index) Sync(ctx context.Context, repositories []hoststate.Repository) error {
	fingerprint := snapshotFingerprint(repositories)

	i.mu.Lock()
	defer i.mu.Unlock()
	if fingerprint == i.fingerprint {
		return nil
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM repositories`); err != nil {
		return err
	}

	repoStatement, err := tx.PrepareContext(ctx, `
		INSERT INTO repositories(id, name, root, first_seen_at, last_seen_at, convention_label, detection_error)
		VALUES(?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer repoStatement.Close()

	sessionStatement, err := tx.PrepareContext(ctx, `
		INSERT INTO sessions(id, repository_id, agent, pid, started_at, last_seen_at, ended_at, active)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer sessionStatement.Close()

	for _, repository := range repositories {
		if _, err := repoStatement.ExecContext(
			ctx,
			repository.ID,
			repository.Name,
			repository.Root,
			repository.FirstSeenAt.UnixNano(),
			repository.LastSeenAt.UnixNano(),
			repository.SpecificationLabel(),
			repository.DetectionError,
		); err != nil {
			return fmt.Errorf("index repository %s: %w", repository.ID, err)
		}

		for _, session := range repository.Sessions {
			var endedAt any
			if session.EndedAt != nil {
				endedAt = session.EndedAt.UnixNano()
			}
			active := 0
			if session.Active {
				active = 1
			}
			if _, err := sessionStatement.ExecContext(
				ctx,
				session.ID,
				repository.ID,
				session.Agent,
				session.PID,
				session.StartedAt.UnixNano(),
				session.LastSeenAt.UnixNano(),
				endedAt,
				active,
			); err != nil {
				return fmt.Errorf("index session %s: %w", session.ID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	i.fingerprint = fingerprint
	return nil
}

func (i *Index) SearchRepositoryIDs(ctx context.Context, query string, limit int) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	pattern := "%" + escapeLike(strings.ToLower(query)) + "%"
	rows, err := i.db.QueryContext(ctx, `
		SELECT r.id
		FROM repositories r
		LEFT JOIN sessions s ON s.repository_id = r.id
		WHERE lower(r.name) LIKE ? ESCAPE '\'
		   OR lower(r.root) LIKE ? ESCAPE '\'
		   OR lower(r.convention_label) LIKE ? ESCAPE '\'
		   OR lower(s.agent) LIKE ? ESCAPE '\'
		GROUP BY r.id
		ORDER BY max(r.last_seen_at) DESC
		LIMIT ?`, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func snapshotFingerprint(repositories []hoststate.Repository) string {
	copyRepositories := append([]hoststate.Repository(nil), repositories...)
	sort.Slice(copyRepositories, func(a, b int) bool { return copyRepositories[a].ID < copyRepositories[b].ID })

	hash := sha256.New()
	for _, repository := range copyRepositories {
		_, _ = fmt.Fprintf(hash, "repo\x00%s\x00%s\x00%s\x00%s\x00%s\n",
			repository.ID,
			repository.Name,
			repository.Root,
			repository.SpecificationLabel(),
			repository.DetectionError,
		)

		sessions := append([]hoststate.Session(nil), repository.Sessions...)
		sort.Slice(sessions, func(a, b int) bool { return sessions[a].ID < sessions[b].ID })
		for _, session := range sessions {
			endedAt := int64(0)
			if session.EndedAt != nil {
				endedAt = session.EndedAt.UnixNano()
			}
			_, _ = fmt.Fprintf(hash, "session\x00%s\x00%s\x00%d\x00%d\x00%d\x00%t\n",
				session.ID,
				session.Agent,
				session.PID,
				session.StartedAt.UnixNano(),
				endedAt,
				session.Active,
			)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}
