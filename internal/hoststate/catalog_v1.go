package hoststate

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sergii/specview/internal/config"
)

type persistedCatalogV1 struct {
	Version      int               `json:"version"`
	Repositories []persistedRepoV1 `json:"repositories"`
}

type persistedRepoV1 struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Root           string               `json:"root"`
	FirstSeenAt    time.Time            `json:"first_seen_at"`
	LastSeenAt     time.Time            `json:"last_seen_at"`
	Convention     config.Convention    `json:"convention"`
	DetectionError string               `json:"detection_error,omitempty"`
	Sessions       []persistedSessionV1 `json:"sessions"`
}

type persistedSessionV1 struct {
	ID         string     `json:"id"`
	Agent      string     `json:"agent"`
	PID        int        `json:"pid"`
	StartedAt  time.Time  `json:"started_at"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	Active     bool       `json:"active"`
}

func migrateCatalogV1(data []byte) ([]Repository, error) {
	var persisted persistedCatalogV1
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("decode host catalog v1: %w", err)
	}
	if persisted.Version != 1 {
		return nil, fmt.Errorf("migrate host catalog v1: unexpected version %d", persisted.Version)
	}

	repositories := make([]Repository, 0, len(persisted.Repositories))
	for _, legacy := range persisted.Repositories {
		repository := Repository{
			ID:             legacy.ID,
			Name:           legacy.Name,
			Root:           filepath.Clean(legacy.Root),
			FirstSeenAt:    legacy.FirstSeenAt,
			LastSeenAt:     legacy.LastSeenAt,
			Convention:     legacy.Convention,
			DetectionError: legacy.DetectionError,
			Sessions:       make([]Session, 0, len(legacy.Sessions)),
		}
		for _, legacySession := range legacy.Sessions {
			processIDs := []int(nil)
			if legacySession.PID > 0 {
				processIDs = []int{legacySession.PID}
			}
			repository.Sessions = append(repository.Sessions, Session{
				ID:           legacySession.ID,
				IdentityKind: SessionIdentityLegacyPID,
				Adapter:      legacyAdapterForAgent(legacySession.Agent),
				Agent:        legacySession.Agent,
				ProcessIDs:   processIDs,
				StartedAt:    legacySession.StartedAt,
				LastSeenAt:   legacySession.LastSeenAt,
				EndedAt:      legacySession.EndedAt,
				Active:       legacySession.Active,
			})
		}
		repositories = append(repositories, repository)
	}
	return repositories, nil
}

func legacyAdapterForAgent(agent string) string {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "codex":
		return "codex"
	case "claude", "claude code":
		return "claude"
	default:
		return "legacy"
	}
}
