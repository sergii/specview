package executionhistory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/hoststate"
)

const SchemaVersion = 1

type Projection struct {
	SchemaVersion int     `json:"schema_version"`
	Hostname      string  `json:"hostname"`
	Entries       []Entry `json:"entries"`
}

type Entry struct {
	RepositoryID   string     `json:"repository_id"`
	RepositoryName string     `json:"repository_name"`
	RepositoryRoot string     `json:"repository_root"`
	SessionID      string     `json:"session_id"`
	IdentityKind   string     `json:"identity_kind"`
	Adapter        string     `json:"adapter"`
	Agent          string     `json:"agent"`
	ProcessIDs     []int      `json:"process_ids"`
	CWD            string     `json:"cwd,omitempty"`
	WorktreeRoot   string     `json:"worktree_root,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	LastSeenAt     time.Time  `json:"last_seen_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	Active         bool       `json:"active"`
}

type Reader struct {
	catalog *hoststate.Catalog
}

func NewReader(catalog *hoststate.Catalog) *Reader {
	return &Reader{catalog: catalog}
}

func (r *Reader) Build(context.Context) (Projection, error) {
	if r == nil || r.catalog == nil {
		return Projection{}, errors.New("execution history catalog is required")
	}
	return Build(r.catalog.Hostname(), r.catalog.Repositories()), nil
}

func (r *Reader) Find(ctx context.Context, repositoryID, sessionID string) (Entry, bool, error) {
	projection, err := r.Build(ctx)
	if err != nil {
		return Entry{}, false, err
	}
	entry, ok := Find(projection, repositoryID, sessionID)
	return entry, ok, nil
}

func Find(projection Projection, repositoryID, sessionID string) (Entry, bool) {
	repositoryID = strings.TrimSpace(repositoryID)
	sessionID = strings.TrimSpace(sessionID)
	if repositoryID == "" || sessionID == "" {
		return Entry{}, false
	}
	for _, entry := range projection.Entries {
		if entry.RepositoryID == repositoryID && entry.SessionID == sessionID {
			return entry, true
		}
	}
	return Entry{}, false
}

func Build(hostname string, repositories []hoststate.Repository) Projection {
	entries := make([]Entry, 0)
	for _, repository := range repositories {
		for _, session := range repository.Sessions {
			processIDs := append([]int(nil), session.ProcessIDs...)
			sort.Ints(processIDs)
			entries = append(entries, Entry{
				RepositoryID:   repository.ID,
				RepositoryName: repository.Name,
				RepositoryRoot: repository.Root,
				SessionID:      session.ID,
				IdentityKind:   session.IdentityKind,
				Adapter:        session.Adapter,
				Agent:          session.Agent,
				ProcessIDs:     processIDs,
				CWD:            session.CWD,
				WorktreeRoot:   session.WorktreeRoot,
				StartedAt:      session.StartedAt,
				LastSeenAt:     session.LastSeenAt,
				EndedAt:        session.EndedAt,
				Active:         session.Active,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].LastSeenAt.Equal(entries[j].LastSeenAt) {
			return entries[i].LastSeenAt.After(entries[j].LastSeenAt)
		}
		if entries[i].RepositoryID != entries[j].RepositoryID {
			return entries[i].RepositoryID < entries[j].RepositoryID
		}
		return entries[i].SessionID < entries[j].SessionID
	})

	return Projection{
		SchemaVersion: SchemaVersion,
		Hostname:      hostname,
		Entries:       entries,
	}
}
