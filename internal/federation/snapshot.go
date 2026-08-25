package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/identity"
)

const (
	// SnapshotSchemaVersion is the frozen H20/H21 wire contract.
	SnapshotSchemaVersion   = 1
	SnapshotSchemaVersionV2 = 2
	CurrentSnapshotVersion  = SnapshotSchemaVersionV2
)

type LocalReader interface {
	ListRepositories(context.Context) (controlplane.ListRepositoriesResult, error)
	GetRepository(context.Context, string) (controlplane.GetRepositoryResult, error)
	ListActiveSessions(context.Context) (controlplane.ListActiveSessionsResult, error)
	GetHostControlPlane(context.Context) (controlplane.GetHostControlPlaneResult, error)
}

type HostSnapshot struct {
	SchemaVersion int                                     `json:"schema_version"`
	HostID        string                                  `json:"host_id"`
	Hostname      string                                  `json:"hostname"`
	ObservedAt    time.Time                               `json:"observed_at"`
	ControlPlane  *controlplane.GetHostControlPlaneResult `json:"control_plane,omitempty"`
	Instances     []RepositoryInstance                    `json:"repository_instances"`
	Warnings      []string                                `json:"warnings,omitempty"`
}

type RepositoryInstance struct {
	InstanceID         string                         `json:"instance_id"`
	SourceRepositoryID string                         `json:"source_repository_id"`
	Name               string                         `json:"name"`
	Root               string                         `json:"root"`
	Fingerprint        identity.RepositoryFingerprint `json:"fingerprint"`
	Active             bool                           `json:"active"`
	Agents             []string                       `json:"agents,omitempty"`
	Sessions           []Session                      `json:"sessions"`
	Worktrees          []Worktree                     `json:"worktrees"`
	Warnings           []string                       `json:"warnings,omitempty"`
}

type Session struct {
	ID           string `json:"id"`
	Adapter      string `json:"adapter"`
	Agent        string `json:"agent"`
	WorktreeRoot string `json:"worktree_root,omitempty"`
	CWD          string `json:"cwd"`
	StartedAt    string `json:"started_at,omitempty"`
}

type Worktree struct {
	Path       string `json:"path"`
	Branch     string `json:"branch,omitempty"`
	Head       string `json:"head,omitempty"`
	Detached   bool   `json:"detached"`
	DirtyCount int    `json:"dirty_count"`
	Upstream   string `json:"upstream,omitempty"`
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
	LastCommit string `json:"last_commit,omitempty"`
}

type Builder struct {
	hostID string
	reader LocalReader
	now    func() time.Time
}

func NewBuilder(hostID string, reader LocalReader) *Builder {
	return &Builder{hostID: strings.TrimSpace(hostID), reader: reader, now: time.Now}
}

func (b *Builder) Build(ctx context.Context) (HostSnapshot, error) {
	if b == nil || b.reader == nil {
		return HostSnapshot{}, errors.New("federation snapshot reader is required")
	}
	if err := identity.ValidateHostID(b.hostID); err != nil {
		return HostSnapshot{}, err
	}

	repositories, err := b.reader.ListRepositories(ctx)
	if err != nil {
		return HostSnapshot{}, err
	}
	sessions, sessionErr := b.reader.ListActiveSessions(ctx)
	controlPlane, controlPlaneErr := b.reader.GetHostControlPlane(ctx)
	if controlPlaneErr != nil {
		return HostSnapshot{}, fmt.Errorf("build Host control plane for federation snapshot: %w", controlPlaneErr)
	}

	snapshot := HostSnapshot{
		SchemaVersion: CurrentSnapshotVersion,
		HostID:        b.hostID,
		Hostname:      repositories.Host,
		ObservedAt:    b.now().UTC(),
		ControlPlane:  &controlPlane,
		Instances:     make([]RepositoryInstance, 0, len(repositories.Repositories)),
		Warnings:      append([]string(nil), repositories.Warnings...),
	}
	if sessionErr != nil {
		snapshot.Warnings = append(snapshot.Warnings, "active sessions unavailable: "+sessionErr.Error())
	}

	sessionsByRepository := make(map[string][]controlplane.SessionSummary)
	if sessionErr == nil {
		for _, session := range sessions.Sessions {
			sessionsByRepository[session.RepositoryID] = append(sessionsByRepository[session.RepositoryID], session)
		}
	}

	for _, repository := range repositories.Repositories {
		detail, detailErr := b.reader.GetRepository(ctx, repository.ID)
		if detailErr != nil {
			return HostSnapshot{}, fmt.Errorf("get repository %s for federation snapshot: %w", repository.ID, detailErr)
		}
		instance, instanceErr := buildInstance(b.hostID, detail.Repository, sessionsByRepository[repository.ID], detail.Warnings)
		if instanceErr != nil {
			return HostSnapshot{}, instanceErr
		}
		snapshot.Instances = append(snapshot.Instances, instance)
	}

	sortSnapshot(&snapshot)
	if err := snapshot.Validate(); err != nil {
		return HostSnapshot{}, err
	}
	return snapshot, nil
}

func (s HostSnapshot) V1() HostSnapshot {
	copySnapshot := s
	copySnapshot.SchemaVersion = SnapshotSchemaVersion
	copySnapshot.ControlPlane = nil
	return copySnapshot
}

func DecodeSnapshot(data []byte) (HostSnapshot, error) {
	var snapshot HostSnapshot
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return HostSnapshot{}, fmt.Errorf("decode federation snapshot: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return HostSnapshot{}, errors.New("decode federation snapshot: multiple JSON values")
		}
		return HostSnapshot{}, fmt.Errorf("decode federation snapshot: %w", err)
	}
	if err := snapshot.Validate(); err != nil {
		return HostSnapshot{}, err
	}
	return snapshot, nil
}

func (s HostSnapshot) Validate() error {
	switch s.SchemaVersion {
	case SnapshotSchemaVersion:
		if s.ControlPlane != nil {
			return errors.New("federation HostSnapshot v1 must not contain control_plane")
		}
	case SnapshotSchemaVersionV2:
		if s.ControlPlane == nil {
			return errors.New("federation HostSnapshot v2 requires control_plane")
		}
		if s.ControlPlane.SchemaVersion != controlplane.SchemaVersion {
			return fmt.Errorf("unsupported Host control-plane schema version %d", s.ControlPlane.SchemaVersion)
		}
		if strings.TrimSpace(s.ControlPlane.Host) != strings.TrimSpace(s.Hostname) {
			return fmt.Errorf("federation HostSnapshot control-plane Host %q does not match hostname %q", s.ControlPlane.Host, s.Hostname)
		}
	default:
		return fmt.Errorf("unsupported federation snapshot version %d", s.SchemaVersion)
	}
	if err := identity.ValidateHostID(strings.TrimSpace(s.HostID)); err != nil {
		return err
	}
	if strings.TrimSpace(s.Hostname) == "" {
		return errors.New("federation snapshot hostname is required")
	}
	if s.ObservedAt.IsZero() {
		return errors.New("federation snapshot observed_at is required")
	}
	seen := make(map[string]struct{}, len(s.Instances))
	for _, instance := range s.Instances {
		if strings.TrimSpace(instance.SourceRepositoryID) == "" {
			return errors.New("federation RepositoryInstance source_repository_id is required")
		}
		if strings.TrimSpace(instance.Root) == "" {
			return errors.New("federation RepositoryInstance root is required")
		}
		expectedID, err := identity.RepositoryInstanceID(s.HostID, instance.Root)
		if err != nil {
			return err
		}
		if instance.InstanceID != expectedID {
			return fmt.Errorf("federation RepositoryInstance %q does not match Host/root identity", instance.InstanceID)
		}
		if _, exists := seen[instance.InstanceID]; exists {
			return fmt.Errorf("duplicate federation RepositoryInstance %q", instance.InstanceID)
		}
		seen[instance.InstanceID] = struct{}{}
	}
	return nil
}

func buildInstance(hostID string, repository controlplane.RepositoryDetail, sessions []controlplane.SessionSummary, warnings []string) (RepositoryInstance, error) {
	instanceID, err := identity.RepositoryInstanceID(hostID, repository.Root)
	if err != nil {
		return RepositoryInstance{}, err
	}

	fingerprint := identity.RepositoryFingerprint{Name: repository.Name}
	instance := RepositoryInstance{
		InstanceID:         instanceID,
		SourceRepositoryID: repository.ID,
		Name:               repository.Name,
		Root:               filepath.Clean(repository.Root),
		Fingerprint:        fingerprint,
		Active:             repository.Active,
		Agents:             append([]string(nil), repository.Agents...),
		Sessions:           make([]Session, 0, len(sessions)),
		Warnings:           append([]string(nil), warnings...),
	}

	if projectID, configWarning := explicitProjectID(repository.Root); projectID != "" {
		instance.Fingerprint.ExplicitID = projectID
	} else if configWarning != "" {
		instance.Warnings = append(instance.Warnings, configWarning)
	}
	if repository.Git != nil {
		instance.Fingerprint.GitRemote = repository.Git.Remote
		instance.Worktrees = make([]Worktree, 0, len(repository.Git.Worktrees))
		for _, worktree := range repository.Git.Worktrees {
			instance.Worktrees = append(instance.Worktrees, Worktree{
				Path:       worktree.Path,
				Branch:     worktree.Branch,
				Head:       worktree.Head,
				Detached:   worktree.Detached,
				DirtyCount: worktree.DirtyCount,
				Upstream:   worktree.Upstream,
				Ahead:      worktree.Ahead,
				Behind:     worktree.Behind,
				LastCommit: worktree.LastCommit,
			})
		}
	}
	if repository.Forge != nil {
		instance.Fingerprint.ForgeProvider = repository.Forge.Provider
		instance.Fingerprint.ForgeRepository = repository.Forge.Repository
	}

	for _, session := range sessions {
		instance.Sessions = append(instance.Sessions, Session{
			ID:           session.ID,
			Adapter:      session.Adapter,
			Agent:        session.Agent,
			WorktreeRoot: session.WorktreeRoot,
			CWD:          session.CWD,
			StartedAt:    session.StartedAt,
		})
	}
	return instance, nil
}

func explicitProjectID(root string) (string, string) {
	cfg, err := config.Load(root)
	if err == nil {
		return cfg.Project.ID, ""
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", ""
	}
	return "", "project identity unavailable: " + err.Error()
}

func sortSnapshot(snapshot *HostSnapshot) {
	if snapshot == nil {
		return
	}
	for i := range snapshot.Instances {
		instance := &snapshot.Instances[i]
		sort.Strings(instance.Agents)
		sort.Slice(instance.Sessions, func(i, j int) bool {
			return instance.Sessions[i].ID < instance.Sessions[j].ID
		})
		sort.Slice(instance.Worktrees, func(i, j int) bool {
			return instance.Worktrees[i].Path < instance.Worktrees[j].Path
		})
		sort.Strings(instance.Warnings)
	}
	sort.Slice(snapshot.Instances, func(i, j int) bool {
		left, right := snapshot.Instances[i], snapshot.Instances[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.InstanceID < right.InstanceID
	})
	sort.Strings(snapshot.Warnings)
}
