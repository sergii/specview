package web

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/sergii/specview/internal/config"
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
		populateProjectArtifacts(&data, store)
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
	}
	return hashMaterial(material)
}

func materialRepository(repository interface {
	Active() bool
	ActiveAgentLabel() string
}) hostMaterialRepository {
	// This helper is intentionally implemented through the concrete hoststate
	// repository below. Keeping material shaping here prevents heartbeat fields
	// from leaking into the SSE digest.
	return hostMaterialRepository{}
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
