package hoststate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/sergii/specview/internal/config"
)

const (
	catalogVersion           = 1
	heartbeatPersistInterval = 30 * time.Second
)

type Observation struct {
	Agent          string
	PID            int
	RepositoryRoot string
}

type Session struct {
	ID         string     `json:"id"`
	Agent      string     `json:"agent"`
	PID        int        `json:"pid"`
	StartedAt  time.Time  `json:"started_at"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	Active     bool       `json:"active"`
}

type Repository struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Root           string            `json:"root"`
	FirstSeenAt    time.Time         `json:"first_seen_at"`
	LastSeenAt     time.Time         `json:"last_seen_at"`
	Convention     config.Convention `json:"convention"`
	DetectionError string            `json:"detection_error,omitempty"`
	Sessions       []Session         `json:"sessions"`
}

func (r Repository) Active() bool {
	for _, session := range r.Sessions {
		if session.Active {
			return true
		}
	}
	return false
}

func (r Repository) ActiveAgentLabel() string {
	seen := make(map[string]struct{})
	var agents []string
	for _, session := range r.Sessions {
		if !session.Active {
			continue
		}
		if _, ok := seen[session.Agent]; ok {
			continue
		}
		seen[session.Agent] = struct{}{}
		agents = append(agents, session.Agent)
	}
	sort.Strings(agents)
	switch len(agents) {
	case 0:
		return ""
	case 1:
		return agents[0]
	default:
		return fmt.Sprintf("%d agents", len(agents))
	}
}

func (r Repository) SpecificationLabel() string {
	if r.DetectionError != "" {
		return "Pattern conflict"
	}
	if !r.Convention.Recognized {
		return "No specification pattern"
	}
	if !r.Convention.Supported {
		return r.Convention.Label + " · detected"
	}
	return r.Convention.Label
}

type persistedCatalog struct {
	Version      int          `json:"version"`
	Repositories []Repository `json:"repositories"`
}

type Detector func(root string) (config.Convention, error)

type Catalog struct {
	path     string
	hostname string
	detect   Detector

	mu              sync.RWMutex
	repos           map[string]*Repository
	lastPersistedAt time.Time
	heartbeatDirty  bool
}

func DefaultStatePath() (string, error) {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "specview", "catalog.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Specview", "catalog.json"), nil
	}
	return filepath.Join(home, ".local", "state", "specview", "catalog.json"), nil
}

func OpenCatalog(path string) (*Catalog, error) {
	return openCatalog(path, config.DetectConvention)
}

func openCatalog(path string, detector Detector) (*Catalog, error) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "localhost"
	}
	c := &Catalog{
		path:     path,
		hostname: hostname,
		detect:   detector,
		repos:    make(map[string]*Repository),
	}
	if path == "" {
		return c, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	var persisted persistedCatalog
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("decode host catalog: %w", err)
	}
	if persisted.Version != catalogVersion {
		return nil, fmt.Errorf("unsupported host catalog version %d", persisted.Version)
	}
	for i := range persisted.Repositories {
		repo := persisted.Repositories[i]
		repo.Root = filepath.Clean(repo.Root)
		c.repos[repo.ID] = &repo
	}
	return c, nil
}

func (c *Catalog) Hostname() string {
	return c.hostname
}

func (c *Catalog) Repositories() []Repository {
	c.mu.RLock()
	defer c.mu.RUnlock()
	repositories := make([]Repository, 0, len(c.repos))
	for _, repo := range c.repos {
		copyRepo := *repo
		copyRepo.Name = repositoryDisplayName(copyRepo.Root)
		copyRepo.Sessions = append([]Session(nil), repo.Sessions...)
		repositories = append(repositories, copyRepo)
	}
	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].LastSeenAt.After(repositories[j].LastSeenAt)
	})
	return repositories
}

func (c *Catalog) Find(id string) (Repository, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	repo, ok := c.repos[id]
	if !ok {
		return Repository{}, false
	}
	copyRepo := *repo
	copyRepo.Name = repositoryDisplayName(copyRepo.Root)
	copyRepo.Sessions = append([]Session(nil), repo.Sessions...)
	return copyRepo, true
}

func (c *Catalog) Observe(observations []Observation, now time.Time) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	activeKeys := make(map[string]struct{})
	changed := false
	materialChanged := false
	heartbeatChanged := false

	for _, observation := range observations {
		root := filepath.Clean(observation.RepositoryRoot)
		if root == "." || root == "" {
			continue
		}
		repoID := repositoryID(root)
		repo, ok := c.repos[repoID]
		if !ok {
			repo = &Repository{
				ID:          repoID,
				Name:        filepath.Base(root),
				Root:        root,
				FirstSeenAt: now,
				LastSeenAt:  now,
			}
			c.repos[repoID] = repo
			changed = true
			materialChanged = true
		}

		convention, err := c.detect(root)
		if err != nil {
			if repo.DetectionError != err.Error() || repo.Convention != (config.Convention{}) {
				repo.DetectionError = err.Error()
				repo.Convention = config.Convention{}
				changed = true
				materialChanged = true
			}
		} else if repo.Convention != convention || repo.DetectionError != "" {
			repo.Convention = convention
			repo.DetectionError = ""
			changed = true
			materialChanged = true
		}

		key := activeSessionKey(repoID, observation.Agent, observation.PID)
		activeKeys[key] = struct{}{}
		sessionIndex := -1
		for i := range repo.Sessions {
			session := &repo.Sessions[i]
			if session.Active && activeSessionKey(repoID, session.Agent, session.PID) == key {
				sessionIndex = i
				break
			}
		}
		if sessionIndex < 0 {
			repo.Sessions = append(repo.Sessions, Session{
				ID:         sessionID(repoID, observation.Agent, observation.PID, now),
				Agent:      observation.Agent,
				PID:        observation.PID,
				StartedAt:  now,
				LastSeenAt: now,
				Active:     true,
			})
			changed = true
			materialChanged = true
		} else {
			session := &repo.Sessions[sessionIndex]
			if !session.LastSeenAt.Equal(now) {
				session.LastSeenAt = now
				changed = true
				heartbeatChanged = true
			}
		}
		if repo.LastSeenAt.Before(now) {
			repo.LastSeenAt = now
			changed = true
			heartbeatChanged = true
		}
	}

	for repoID, repo := range c.repos {
		for i := range repo.Sessions {
			session := &repo.Sessions[i]
			if !session.Active {
				continue
			}
			if _, ok := activeKeys[activeSessionKey(repoID, session.Agent, session.PID)]; ok {
				continue
			}
			ended := now
			session.Active = false
			session.EndedAt = &ended
			changed = true
			materialChanged = true
		}
	}

	switch {
	case materialChanged:
		if err := c.saveLocked(now); err != nil {
			return false, err
		}
		c.heartbeatDirty = false
	case heartbeatChanged:
		c.heartbeatDirty = true
		if c.lastPersistedAt.IsZero() || !now.Before(c.lastPersistedAt.Add(heartbeatPersistInterval)) {
			if err := c.saveLocked(now); err != nil {
				return false, err
			}
			c.heartbeatDirty = false
		}
	}
	return changed, nil
}

// Flush persists any coalesced heartbeat state. Material lifecycle changes are
// already persisted synchronously by Observe, so a clean Flush is a no-op.
func (c *Catalog) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.heartbeatDirty {
		return nil
	}
	if err := c.saveLocked(time.Now()); err != nil {
		return err
	}
	c.heartbeatDirty = false
	return nil
}

func (c *Catalog) saveLocked(persistedAt time.Time) error {
	if c.path == "" {
		c.lastPersistedAt = persistedAt
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	repositories := make([]Repository, 0, len(c.repos))
	for _, repo := range c.repos {
		repositories = append(repositories, *repo)
	}
	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].LastSeenAt.After(repositories[j].LastSeenAt)
	})
	data, err := json.MarshalIndent(persistedCatalog{
		Version:      catalogVersion,
		Repositories: repositories,
	}, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return err
	}
	c.lastPersistedAt = persistedAt
	return nil
}

func repositoryDisplayName(root string) string {
	clean := filepath.Clean(root)
	base := filepath.Base(clean)
	if base == "." || base == "" || base == string(filepath.Separator) {
		return base
	}
	parent := filepath.Base(filepath.Dir(clean))
	if parent == "." || parent == "" || parent == string(filepath.Separator) {
		return base
	}
	return parent + "/" + base
}

func repositoryID(root string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(root)))
	return "repo-" + hex.EncodeToString(sum[:8])
}

func activeSessionKey(repoID, agent string, pid int) string {
	return fmt.Sprintf("%s|%s|%d", repoID, agent, pid)
}

func sessionID(repoID, agent string, pid int, started time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%d", repoID, agent, pid, started.UnixNano())))
	return "session-" + hex.EncodeToString(sum[:8])
}
