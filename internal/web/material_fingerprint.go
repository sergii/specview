package web

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/evidence"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

type hostMaterialSession struct {
	ID        string
	Agent     string
	PID       int
	StartedAt time.Time
	EndedAt   *time.Time
	Active    bool
}

type hostMaterialRepository struct {
	ID             string
	Name           string
	Root           string
	Convention     config.Convention
	DetectionError string
	Active         bool
	ActiveAgent    string
	Sessions       []hostMaterialSession
}

type materialEvidenceRecord struct {
	Path       string
	Error      string
	ID         string
	WorkItemID string
	Revision   string
	Check      string
	Kind       evidence.Kind
	Provider   string
	Result     evidence.Result
	ObservedAt time.Time
	Summary    string
	Metrics    map[string]float64
}

type projectMaterialState struct {
	RepositoryID      string
	RepositoryName    string
	RepositoryRoot    string
	Active            bool
	ActiveAgent       string
	Sessions          []hostMaterialSession
	SourceControl     sourcecontrol.RepositoryContext
	Convention        config.Convention
	DetectionError    string
	Unsupported       bool
	New               []specs.Artifact
	InProgress        []specs.Artifact
	Done              []specs.Artifact
	Invalid           []specs.Artifact
	Total             int
	SpecificationRoot string
	Acceptance        config.Acceptance
	Evidence          []materialEvidenceRecord
}

func (s *HostServer) materialFingerprint(ctx context.Context, scope, projectID string) (string, error) {
	switch scope {
	case "", "host":
		return s.hostMaterialFingerprint()
	case "project":
		return s.projectMaterialFingerprint(ctx, projectID)
	default:
		return "", fmt.Errorf("unsupported live projection scope %q", scope)
	}
}

func (s *HostServer) hostMaterialFingerprint() (string, error) {
	repositories := s.catalog.Repositories()
	material := make([]hostMaterialRepository, 0, len(repositories))
	for _, repository := range repositories {
		material = append(material, materialRepository(repository))
	}
	sort.Slice(material, func(i, j int) bool { return material[i].ID < material[j].ID })
	return hashMaterial(material)
}

func (s *HostServer) projectMaterialFingerprint(ctx context.Context, projectID string) (string, error) {
	repository, ok := s.catalog.Find(projectID)
	if !ok {
		return "", fmt.Errorf("repository %q not found", projectID)
	}

	data, store, err := s.projectStore(repository)
	if err != nil {
		return "", err
	}
	if store != nil {
		for _, item := range store.All() {
			if !item.IsBoardItem() {
				continue
			}
			data.Total++
			if item.Error != "" {
				data.Invalid = append(data.Invalid, item)
				continue
			}
			switch item.Status {
			case specs.StatusNew:
				data.New = append(data.New, item)
			case specs.StatusInProgress:
				data.InProgress = append(data.InProgress, item)
			case specs.StatusDone:
				data.Done = append(data.Done, item)
			}
		}
	}

	sourceContext, err := s.sourceControl.Inspect(ctx, repository.Root)
	if err != nil {
		return "", err
	}
	sourceContext.Git.Worktrees = append([]sourcecontrol.Worktree(nil), sourceContext.Git.Worktrees...)
	sort.Slice(sourceContext.Git.Worktrees, func(i, j int) bool {
		return sourceContext.Git.Worktrees[i].Path < sourceContext.Git.Worktrees[j].Path
	})
	sourceContext.Provider.PullRequests = append([]sourcecontrol.PullRequest(nil), sourceContext.Provider.PullRequests...)
	sort.Slice(sourceContext.Provider.PullRequests, func(i, j int) bool {
		return sourceContext.Provider.PullRequests[i].Number < sourceContext.Provider.PullRequests[j].Number
	})

	acceptanceConfig, evidenceRecords, err := projectAcceptanceMaterial(repository.Root)
	if err != nil {
		return "", err
	}

	repositoryMaterial := materialRepository(repository)
	material := projectMaterialState{
		RepositoryID:      repository.ID,
		RepositoryName:    repository.Name,
		RepositoryRoot:    repository.Root,
		Active:            repositoryMaterial.Active,
		ActiveAgent:       repositoryMaterial.ActiveAgent,
		Sessions:          repositoryMaterial.Sessions,
		SourceControl:     sourceContext,
		Convention:        data.Convention,
		DetectionError:    data.DetectionError,
		Unsupported:       data.Unsupported,
		New:               stableArtifacts(data.New),
		InProgress:        stableArtifacts(data.InProgress),
		Done:              stableArtifacts(data.Done),
		Invalid:           stableArtifacts(data.Invalid),
		Total:             data.Total,
		SpecificationRoot: data.SpecificationRoot,
		Acceptance:        acceptanceConfig,
		Evidence:          evidenceRecords,
	}
	return hashMaterial(material)
}

func projectAcceptanceMaterial(repositoryRoot string) (config.Acceptance, []materialEvidenceRecord, error) {
	projectRoot := filepath.Clean(repositoryRoot)
	acceptanceConfig := config.Acceptance{}
	configPath := filepath.Join(repositoryRoot, config.FileName)
	if _, err := os.Stat(configPath); err == nil {
		cfg, err := config.Load(repositoryRoot)
		if err != nil {
			return config.Acceptance{}, nil, err
		}
		projectRoot = cfg.ResolveProjectRoot(repositoryRoot)
		acceptanceConfig = cfg.Acceptance
	} else if !errors.Is(err, os.ErrNotExist) {
		return config.Acceptance{}, nil, err
	}

	records, err := evidence.NewNativeAdapter(filepath.Join(projectRoot, ".specview", "evidence")).Scan()
	if err != nil {
		return config.Acceptance{}, nil, err
	}
	material := make([]materialEvidenceRecord, 0, len(records))
	for _, record := range records {
		material = append(material, materialEvidenceRecord{
			Path:       record.Path,
			Error:      record.Error,
			ID:         record.ID,
			WorkItemID: record.WorkItemID,
			Revision:   record.Revision,
			Check:      record.Check,
			Kind:       record.Kind,
			Provider:   record.Provider,
			Result:     record.Result,
			ObservedAt: record.ObservedAt,
			Summary:    record.Summary,
			Metrics:    record.Metrics,
		})
	}
	sort.Slice(material, func(i, j int) bool {
		if material[i].Path == material[j].Path {
			return material[i].ID < material[j].ID
		}
		return material[i].Path < material[j].Path
	})
	return acceptanceConfig, material, nil
}

func materialRepository(repository hoststate.Repository) hostMaterialRepository {
	sessions := make([]hostMaterialSession, 0, len(repository.Sessions))
	for _, session := range repository.Sessions {
		sessions = append(sessions, hostMaterialSession{
			ID:        session.ID,
			Agent:     session.Agent,
			PID:       session.PID,
			StartedAt: session.StartedAt,
			EndedAt:   session.EndedAt,
			Active:    session.Active,
		})
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].ID < sessions[j].ID })
	return hostMaterialRepository{
		ID:             repository.ID,
		Name:           repository.Name,
		Root:           repository.Root,
		Convention:     repository.Convention,
		DetectionError: repository.DetectionError,
		Active:         repository.Active(),
		ActiveAgent:    repository.ActiveAgentLabel(),
		Sessions:       sessions,
	}
}

func stableArtifacts(items []specs.Artifact) []specs.Artifact {
	stable := append([]specs.Artifact(nil), items...)
	sort.Slice(stable, func(i, j int) bool {
		if stable[i].Path == stable[j].Path {
			return stable[i].ID < stable[j].ID
		}
		return stable[i].Path < stable[j].Path
	})
	return stable
}

func hashMaterial(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", sum[:]), nil
}
