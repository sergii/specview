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
	"strings"
	"sync"
	"time"

	"github.com/sergii/specview/internal/config"
)

const (
	catalogVersion           = 2
	heartbeatPersistInterval = 30 * time.Second

	SessionIdentityLogical   = "logical"
	SessionIdentityLegacyPID = "legacy_pid"
)

type Observation struct {
	Agent          string
	PID            int
	RepositoryRoot string
}

type Session struct {
	ID           string     `json:"id"`
	IdentityKind string     `json:"identity_kind"`
	Adapter      string     `json:"adapter"`
	Agent        string     `json:"agent"`
	ProcessIDs   []int      `json:"process_ids"`
	CWD          string     `json:"cwd,omitempty"`
	WorktreeRoot string     `json:"worktree_root,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	LastSeenAt   time.Time  `json:"last_seen_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Active       bool       `json:"active"`
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
	materialDirty   bool
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

	var header struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode host catalog header: %w", err)
	}

	var repositories []Repository
	switch header.Version {
	case catalogVersion:
		var persisted persistedCatalog
		if err := json.Unmarshal(data, &persisted); err != nil {
			return nil, fmt.Errorf("decode host catalog v%d: %w", catalogVersion, err)
		}
		repositories = persisted.Repositories
	case 1:
		repositories, err = migrateCatalogV1(data)
		if err != nil {
			return nil, err
		}
		c.materialDirty = true
	default:
		return nil, fmt.Errorf("unsupported host catalog version %d", header.Version)
	}

	for i := range repositories {
		repo := repositories[i]
		repo.Root = filepath.Clean(repo.Root)
		for j := range repo.Sessions {
			if err := normalizePersistedSession(&repo.Sessions[j]); err != nil {
				return nil, fmt.Errorf("repository %s session %d: %w", repo.ID, j, err)
			}
		}
		c.repos[repo.ID] = &repo
	}
	return c, nil
}

func normalizePersistedSession(session *Session) error {
	if session == nil {
		return errors.New("session is required")
	}
	session.ID = strings.TrimSpace(session.ID)
	session.IdentityKind = strings.TrimSpace(session.IdentityKind)
	session.Adapter = strings.TrimSpace(session.Adapter)
	session.Agent = strings.TrimSpace(session.Agent)
	if session.ID == "" {
		return errors.New("session id is required")
	}
	switch session.IdentityKind {
	case SessionIdentityLogical, SessionIdentityLegacyPID:
	default:
		return fmt.Errorf("unsupported identity_kind %q", session.IdentityKind)
	}
	if session.Adapter == "" {
		return errors.New("session adapter is required")
	}
	if session.Agent == "" {
		return errors.New("session agent is required")
	}
	session.ProcessIDs = normalizeProcessIDs(session.ProcessIDs)
	if session.CWD != "" {
		session.CWD = filepath.Clean(session.CWD)
	}
	if session.WorktreeRoot != "" {
		session.WorktreeRoot = filepath.Clean(session.WorktreeRoot)
	}
	return nil
}

func (c *Catalog) Hostname() string {
	return c.hostname
}

func (c *Catalog) Repositories() []Repository {
	c.mu.RLock()
	defer c.mu.RUnlock()
	repositories := make([]Repository, 0, len(c.repos))
	for _, repo := range c.repos {
		repositories = append(repositories, copyRepository(*repo))
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
	return copyRepository(*repo), true
}

func copyRepository(repo Repository) Repository {
	copyRepo := repo
	copyRepo.Name = repositoryDisplayName(copyRepo.Root)
	copyRepo.Sessions = make([]Session, len(repo.Sessions))
	for i := range repo.Sessions {
		copyRepo.Sessions[i] = repo.Sessions[i]
		copyRepo.Sessions[i].ProcessIDs = append([]int(nil), repo.Sessions[i].ProcessIDs...)
	}
	return copyRepo
}

// Observe is the legacy process-shaped compatibility path. The default product
// runtime persists logical ExecutionSessions through ObserveExecutions instead.
func (c *Catalog) Observe(observations []Observation, now time.Time) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	activeKeys := make(map[string]struct{})
	changed := false
	materialChanged := false
	heartbeatChanged := false

	for _, observation := range observations {
		root := filepath.Clean(observation.RepositoryRoot)
		if root == "." || root == "" || observation.PID <= 0 || strings.TrimSpace(observation.Agent) == "" {
			continue
		}
		repo, repoChanged := c.ensureRepositoryLocked(root, now)
		if repoChanged {
			changed = true
			materialChanged = true
		}

		key := activeProcessKey(repo.ID, observation.Agent, observation.PID)
		activeKeys[key] = struct{}{}
		sessionIndex := findActiveSessionByProcess(repo.Sessions, observation.Agent, observation.PID)
		if sessionIndex < 0 {
			repo.Sessions = append(repo.Sessions, Session{
				ID:           sessionID(repo.ID, observation.Agent, observation.PID, now),
				IdentityKind: SessionIdentityLegacyPID,
				Adapter:      legacyAdapterForAgent(observation.Agent),
				Agent:        strings.TrimSpace(observation.Agent),
				ProcessIDs:   []int{observation.PID},
				StartedAt:    now,
				LastSeenAt:   now,
				Active:       true,
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
			if !session.Active || sessionSeenByProcesses(repoID, *session, activeKeys) {
				continue
			}
			ended := now
			session.Active = false
			session.EndedAt = &ended
			changed = true
			materialChanged = true
		}
	}

	return c.finishObservationLocked(changed, materialChanged, heartbeatChanged, now)
}

func (c *Catalog) ensureRepositoryLocked(root string, now time.Time) (*Repository, bool) {
	repoID := repositoryID(root)
	repo, ok := c.repos[repoID]
	changed := false
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
	}

	convention, err := c.detect(root)
	if err != nil {
		if repo.DetectionError != err.Error() || repo.Convention != (config.Convention{}) {
			repo.DetectionError = err.Error()
			repo.Convention = config.Convention{}
			changed = true
		}
	} else if repo.Convention != convention || repo.DetectionError != "" {
		repo.Convention = convention
		repo.DetectionError = ""
		changed = true
	}
	return repo, changed
}

func (c *Catalog) finishObservationLocked(changed, materialChanged, heartbeatChanged bool, now time.Time) (bool, error) {
	if materialChanged {
		c.materialDirty = true
	}
	if heartbeatChanged {
		c.heartbeatDirty = true
	}

	switch {
	case c.materialDirty:
		if err := c.saveLocked(now); err != nil {
			return false, err
		}
		c.materialDirty = false
		c.heartbeatDirty = false
	case c.heartbeatDirty && (c.lastPersistedAt.IsZero() || !now.Before(c.lastPersistedAt.Add(heartbeatPersistInterval))):
		if err := c.saveLocked(now); err != nil {
			return false, err
		}
		c.heartbeatDirty = false
	}
	return changed, nil
}

// Flush persists pending catalog state. Lifecycle changes normally persist
// synchronously during observation; Flush also retries a failed material save.
func (c *Catalog) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.materialDirty && !c.heartbeatDirty {
		return nil
	}
	if err := c.saveLocked(time.Now()); err != nil {
		return err
	}
	c.materialDirty = false
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
		copyRepo := copyRepository(*repo)
		sort.Slice(copyRepo.Sessions, func(i, j int) bool { return copyRepo.Sessions[i].ID < copyRepo.Sessions[j].ID })
		repositories = append(repositories, copyRepo)
	}
	sort.Slice(repositories, func(i, j int) bool { return repositories[i].ID < repositories[j].ID })
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

func activeProcessKey(repoID, agent string, pid int) string {
	return fmt.Sprintf("%s|%s|%d", repoID, strings.TrimSpace(agent), pid)
}

func findActiveSessionByProcess(sessions []Session, agent string, pid int) int {
	for i := range sessions {
		if sessions[i].Active && sessions[i].Agent == strings.TrimSpace(agent) && sessionContainsPID(sessions[i], pid) {
			return i
		}
	}
	return -1
}

func sessionContainsPID(session Session, pid int) bool {
	for _, processID := range session.ProcessIDs {
		if processID == pid {
			return true
		}
	}
	return false
}

func sessionSeenByProcesses(repoID string, session Session, active map[string]struct{}) bool {
	for _, pid := range session.ProcessIDs {
		if _, ok := active[activeProcessKey(repoID, session.Agent, pid)]; ok {
			return true
		}
	}
	return false
}

func normalizeProcessIDs(processIDs []int) []int {
	seen := make(map[int]struct{}, len(processIDs))
	result := make([]int, 0, len(processIDs))
	for _, pid := range processIDs {
		if pid <= 0 {
			continue
		}
		if _, exists := seen[pid]; exists {
			continue
		}
		seen[pid] = struct{}{}
		result = append(result, pid)
	}
	sort.Ints(result)
	return result
}

func sessionID(repoID, agent string, pid int, started time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%d", repoID, agent, pid, started.UnixNano())))
	return "session-" + hex.EncodeToString(sum[:8])
}
