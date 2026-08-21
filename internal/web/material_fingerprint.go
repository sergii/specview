package web

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/sergii/specview/internal/config"
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

type projectMaterialExecutionSession struct {
	Adapter        string
	ID             string
	Agent          string
	CWD            string
	RepositoryRoot string
	WorktreeRoot   string
	ProcessCount   int
	StartedAt      time.Time
}

type projectMaterialExecution struct {
	Remote    string
	Worktrees []hoststate.Worktree
	Sessions  []projectMaterialExecutionSession
	Error     string
}

type projectMaterialState struct {
	RepositoryID      string
	RepositoryName    string
	RepositoryRoot    string
	Active            bool
	ActiveAgent       string
	Execution         projectMaterialExecution
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
		material = append(material, hostMaterialRepository{
			ID:             repository.ID,
			Name:           repository.Name,
			Root:           repository.Root,
			Convention:     repository.Convention,
			DetectionError: repository.DetectionError,
			Active:         repository.Active(),
			ActiveAgent:    repository.ActiveAgentLabel(),
			Sessions:       sessions,
		})
	}
	sort.Slice(material, func(i, j int) bool { return material[i].ID < material[j].ID })
	return hashMaterial(material)
}

func (s *HostServer) projectMaterialFingerprint(ctx context.Context, projectID string) (string, error) {
	repository, ok := s.catalog.Find(projectID)
	if !ok {
		return "", fmt.Errorf("repository %q not found", projectID)
	}
	data, err := s.loadProject(ctx, repository)
	if err != nil {
		return "", err
	}

	executionSessions := make([]projectMaterialExecutionSession, 0, len(data.Execution.Sessions))
	for _, session := range data.Execution.Sessions {
		executionSessions = append(executionSessions, projectMaterialExecutionSession{
			Adapter:        session.Adapter,
			ID:             session.ID,
			Agent:          session.Agent,
			CWD:            session.CWD,
			RepositoryRoot: session.RepositoryRoot,
			WorktreeRoot:   session.WorktreeRoot,
			ProcessCount:   len(session.ProcessIDs),
			StartedAt:      session.StartedAt,
		})
	}
	sort.Slice(executionSessions, func(i, j int) bool { return executionSessions[i].ID < executionSessions[j].ID })

	executionWorktrees := append([]hoststate.Worktree(nil), data.Execution.Worktrees...)
	sort.Slice(executionWorktrees, func(i, j int) bool { return executionWorktrees[i].Path < executionWorktrees[j].Path })

	sourceContext := data.SourceControl
	sourceContext.Git.Worktrees = append([]sourcecontrol.Worktree(nil), sourceContext.Git.Worktrees...)
	sort.Slice(sourceContext.Git.Worktrees, func(i, j int) bool {
		return sourceContext.Git.Worktrees[i].Path < sourceContext.Git.Worktrees[j].Path
	})
	sourceContext.Provider.PullRequests = append([]sourcecontrol.PullRequest(nil), sourceContext.Provider.PullRequests...)
	sort.Slice(sourceContext.Provider.PullRequests, func(i, j int) bool {
		return sourceContext.Provider.PullRequests[i].Number < sourceContext.Provider.PullRequests[j].Number
	})

	material := projectMaterialState{
		RepositoryID:   data.Repo.ID,
		RepositoryName: data.Repo.Name,
		RepositoryRoot: data.Repo.Root,
		Active:         data.Repo.Active(),
		ActiveAgent:    data.Repo.ActiveAgentLabel(),
		Execution: projectMaterialExecution{
			Remote:    data.Execution.Remote,
			Worktrees: executionWorktrees,
			Sessions:  executionSessions,
			Error:     data.Execution.Error,
		},
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
