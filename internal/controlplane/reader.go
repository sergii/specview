package controlplane

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

type Reader struct {
	statePath       string
	executionSource hoststate.ExecutionSource
	sourceControl   sourcecontrol.Source
}

func NewReader(statePath string, executionSource hoststate.ExecutionSource, sourceControl sourcecontrol.Source) *Reader {
	if executionSource == nil {
		executionSource = hoststate.DefaultExecutionRegistry()
	}
	if sourceControl == nil {
		sourceControl = sourcecontrol.DefaultService()
	}
	return &Reader{
		statePath:       statePath,
		executionSource: executionSource,
		sourceControl:   sourceControl,
	}
}

func (r *Reader) ListRepositories(ctx context.Context) (ListRepositoriesResult, error) {
	catalog, err := hoststate.OpenCatalog(r.statePath)
	if err != nil {
		return ListRepositoriesResult{}, err
	}

	result := ListRepositoriesResult{
		SchemaVersion: SchemaVersion,
		Host:          catalog.Hostname(),
	}
	byRoot := make(map[string]RepositorySummary)
	for _, repository := range catalog.Repositories() {
		summary := repositorySummary(repository)
		byRoot[normalizeRoot(repository.Root)] = summary
	}

	sessions, sessionErr := r.executionSource.Sessions()
	if sessionErr != nil {
		result.Warnings = append(result.Warnings, "live execution state unavailable: "+sessionErr.Error())
	} else {
		agentsByRoot := make(map[string]map[string]struct{})
		for _, session := range sessions {
			root := normalizeRoot(session.RepositoryRoot)
			if root == "" {
				continue
			}
			summary, ok := byRoot[root]
			if !ok {
				summary = transientRepositorySummary(session.RepositoryRoot)
			}
			summary.Active = true
			byRoot[root] = summary
			if agentsByRoot[root] == nil {
				agentsByRoot[root] = make(map[string]struct{})
			}
			if strings.TrimSpace(session.Agent) != "" {
				agentsByRoot[root][session.Agent] = struct{}{}
			}
		}
		for root, agents := range agentsByRoot {
			summary := byRoot[root]
			summary.Agents = sortedSet(agents)
			byRoot[root] = summary
		}
		// Persisted session Active flags are historical observer state. Once a
		// live scan succeeds, only sessions observed by the live execution source
		// are considered active for this read.
		for root, summary := range byRoot {
			if _, ok := agentsByRoot[root]; !ok {
				summary.Active = false
				summary.Agents = nil
				byRoot[root] = summary
			}
		}
	}

	result.Repositories = make([]RepositorySummary, 0, len(byRoot))
	for _, repository := range byRoot {
		result.Repositories = append(result.Repositories, repository)
	}
	sort.Slice(result.Repositories, func(i, j int) bool {
		left, right := result.Repositories[i], result.Repositories[j]
		if left.Active != right.Active {
			return left.Active
		}
		if left.LastSeenAt != right.LastSeenAt {
			return left.LastSeenAt > right.LastSeenAt
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.ID < right.ID
	})
	return result, nil
}

func (r *Reader) GetRepository(ctx context.Context, repositoryID string) (GetRepositoryResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return GetRepositoryResult{}, err
	}

	result := GetRepositoryResult{
		SchemaVersion: SchemaVersion,
		Host:          catalog.Hostname(),
		Repository: RepositoryDetail{
			RepositorySummary: repositorySummary(repository),
		},
	}

	sessions, sessionErr := r.executionSource.Sessions()
	if sessionErr != nil {
		result.Warnings = append(result.Warnings, "live execution state unavailable: "+sessionErr.Error())
	} else {
		agents := make(map[string]struct{})
		for _, session := range sessions {
			if normalizeRoot(session.RepositoryRoot) != normalizeRoot(repository.Root) {
				continue
			}
			result.Repository.Active = true
			if strings.TrimSpace(session.Agent) != "" {
				agents[session.Agent] = struct{}{}
			}
		}
		result.Repository.Agents = sortedSet(agents)
		if len(agents) == 0 {
			result.Repository.Active = false
		}
	}

	sourceContext, sourceErr := r.sourceControl.Inspect(ctx, repository.Root)
	if sourceErr != nil {
		result.Warnings = append(result.Warnings, "source-control context unavailable: "+sourceErr.Error())
		return result, nil
	}
	git, forge := sourceSummaries(sourceContext)
	result.Repository.Git = &git
	if forge != nil {
		result.Repository.Forge = forge
	}
	return result, nil
}

func (r *Reader) ListActiveSessions(ctx context.Context) (ListActiveSessionsResult, error) {
	catalog, err := hoststate.OpenCatalog(r.statePath)
	if err != nil {
		return ListActiveSessionsResult{}, err
	}
	result := ListActiveSessionsResult{
		SchemaVersion: SchemaVersion,
		Host:          catalog.Hostname(),
	}

	sessions, err := r.executionSource.Sessions()
	if err != nil {
		return result, err
	}
	result.Sessions = make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		startedAt := ""
		if !session.StartedAt.IsZero() {
			startedAt = session.StartedAt.UTC().Format(time.RFC3339Nano)
		}
		result.Sessions = append(result.Sessions, SessionSummary{
			ID:             session.ID,
			Adapter:        session.Adapter,
			Agent:          session.Agent,
			RepositoryID:   hoststate.RepositoryIDForRoot(session.RepositoryRoot),
			RepositoryRoot: filepath.Clean(session.RepositoryRoot),
			WorktreeRoot:   cleanOptionalPath(session.WorktreeRoot),
			CWD:            filepath.Clean(session.CWD),
			ProcessIDs:     append([]int(nil), session.ProcessIDs...),
			StartedAt:      startedAt,
		})
	}
	sort.Slice(result.Sessions, func(i, j int) bool {
		left, right := result.Sessions[i], result.Sessions[j]
		if left.RepositoryID != right.RepositoryID {
			return left.RepositoryID < right.RepositoryID
		}
		if left.WorktreeRoot != right.WorktreeRoot {
			return left.WorktreeRoot < right.WorktreeRoot
		}
		return left.ID < right.ID
	})
	return result, nil
}

func (r *Reader) ListWorktrees(ctx context.Context, repositoryID string) (ListWorktreesResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return ListWorktreesResult{}, err
	}

	result := ListWorktreesResult{
		SchemaVersion:  SchemaVersion,
		Host:           catalog.Hostname(),
		RepositoryID:   repository.ID,
		RepositoryName: repository.Name,
	}
	sourceContext, sourceErr := r.sourceControl.Inspect(ctx, repository.Root)
	if sourceErr != nil {
		result.Warnings = append(result.Warnings, "source-control context unavailable: "+sourceErr.Error())
		return result, nil
	}
	result.Worktrees = worktreeSummaries(sourceContext.Git.Worktrees)
	return result, nil
}

func repositorySummary(repository hoststate.Repository) RepositorySummary {
	summary := RepositorySummary{
		ID:            repository.ID,
		Name:          repository.Name,
		Root:          filepath.Clean(repository.Root),
		Active:        repository.Active(),
		SpecAdapter:   repository.Convention.Adapter,
		SpecLabel:     repository.Convention.Label,
		SpecDetected:  repository.Convention.Recognized,
		SpecSupported: repository.Convention.Supported,
	}
	if !repository.FirstSeenAt.IsZero() {
		summary.FirstSeenAt = repository.FirstSeenAt.UTC().Format(time.RFC3339Nano)
	}
	if !repository.LastSeenAt.IsZero() {
		summary.LastSeenAt = repository.LastSeenAt.UTC().Format(time.RFC3339Nano)
	}
	return summary
}

func transientRepositorySummary(root string) RepositorySummary {
	cleanRoot := filepath.Clean(root)
	return RepositorySummary{
		ID:     hoststate.RepositoryIDForRoot(cleanRoot),
		Name:   hoststate.RepositoryDisplayNameForRoot(cleanRoot),
		Root:   cleanRoot,
		Active: true,
	}
}

func sourceSummaries(repository sourcecontrol.RepositoryContext) (GitSummary, *ForgeSummary) {
	git := GitSummary{
		Remote:    repository.Git.Remote,
		Worktrees: worktreeSummaries(repository.Git.Worktrees),
	}
	if repository.Provider.Name == "" && !repository.Provider.Matched && !repository.Provider.Available && repository.Provider.Repository == "" && repository.Provider.Error == "" {
		return git, nil
	}
	forge := &ForgeSummary{
		Provider:   repository.Provider.Name,
		Matched:    repository.Provider.Matched,
		Available:  repository.Provider.Available,
		Repository: repository.Provider.Repository,
		WebURL:     repository.Provider.WebURL,
		Error:      repository.Provider.Error,
	}
	for _, pullRequest := range repository.Provider.PullRequests {
		forge.PullRequests = append(forge.PullRequests, PullRequestSummary{
			Number: pullRequest.Number,
			Title:  pullRequest.Title,
			URL:    pullRequest.URL,
			State:  pullRequest.State,
			Draft:  pullRequest.Draft,
			Base:   pullRequest.Base,
			Head:   pullRequest.Head,
			Checks: CheckSummary{
				Total:   pullRequest.Checks.Total,
				Passed:  pullRequest.Checks.Passed,
				Failed:  pullRequest.Checks.Failed,
				Pending: pullRequest.Checks.Pending,
				Skipped: pullRequest.Checks.Skipped,
			},
		})
	}
	return git, forge
}

func worktreeSummaries(worktrees []sourcecontrol.Worktree) []WorktreeSummary {
	result := make([]WorktreeSummary, 0, len(worktrees))
	for _, worktree := range worktrees {
		result = append(result, WorktreeSummary{
			Path:       filepath.Clean(worktree.Path),
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
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result
}

func normalizeRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	return filepath.Clean(root)
}

func cleanOptionalPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Clean(path)
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
