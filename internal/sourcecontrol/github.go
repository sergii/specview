package sourcecontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
)

type commandRunner func(context.Context, string, ...string) ([]byte, error)

type GitHubCLIAdapter struct {
	lookPath func(string) (string, error)
	run      commandRunner
}

func NewGitHubCLIAdapter() *GitHubCLIAdapter {
	return &GitHubCLIAdapter{
		lookPath: exec.LookPath,
		run:      runCommand,
	}
}

func (a *GitHubCLIAdapter) Name() string { return "GitHub" }

func (a *GitHubCLIAdapter) Supports(remote string) bool {
	_, _, ok := parseGitHubRemote(remote)
	return ok
}

func (a *GitHubCLIAdapter) Inspect(ctx context.Context, gitContext GitContext) ProviderContext {
	repository, webURL, ok := parseGitHubRemote(gitContext.Remote)
	result := ProviderContext{
		Name:       a.Name(),
		Matched:    ok,
		Repository: repository,
		WebURL:     webURL,
	}
	if !ok {
		return result
	}
	if _, err := a.lookPath("gh"); err != nil {
		result.Error = "GitHub CLI (gh) is not installed"
		return result
	}

	output, err := a.run(ctx, "gh",
		"pr", "list",
		"--repo", repository,
		"--state", "open",
		"--limit", "100",
		"--json", "number,title,url,state,isDraft,baseRefName,headRefName,statusCheckRollup",
	)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	var pullRequests []githubPullRequest
	if err := json.Unmarshal(output, &pullRequests); err != nil {
		result.Error = fmt.Sprintf("decode GitHub pull requests: %v", err)
		return result
	}

	branches := make(map[string]struct{})
	for _, worktree := range gitContext.Worktrees {
		if !worktree.Detached && worktree.Branch != "" {
			branches[worktree.Branch] = struct{}{}
		}
	}
	for _, pullRequest := range pullRequests {
		if _, ok := branches[pullRequest.HeadRefName]; !ok {
			continue
		}
		result.PullRequests = append(result.PullRequests, PullRequest{
			Number: pullRequest.Number,
			Title:  pullRequest.Title,
			URL:    pullRequest.URL,
			State:  pullRequest.State,
			Draft:  pullRequest.IsDraft,
			Base:   pullRequest.BaseRefName,
			Head:   pullRequest.HeadRefName,
			Checks: summarizeChecks(pullRequest.StatusCheckRollup),
		})
	}
	sort.Slice(result.PullRequests, func(i, j int) bool {
		return result.PullRequests[i].Number < result.PullRequests[j].Number
	})
	result.Available = true
	return result
}

type githubPullRequest struct {
	Number            int              `json:"number"`
	Title             string           `json:"title"`
	URL               string           `json:"url"`
	State             string           `json:"state"`
	IsDraft           bool             `json:"isDraft"`
	BaseRefName       string           `json:"baseRefName"`
	HeadRefName       string           `json:"headRefName"`
	StatusCheckRollup []map[string]any `json:"statusCheckRollup"`
}

func summarizeChecks(items []map[string]any) CheckSummary {
	summary := CheckSummary{Total: len(items)}
	for _, item := range items {
		verdict := firstString(item, "conclusion", "state", "status")
		switch strings.ToUpper(verdict) {
		case "SUCCESS":
			summary.Passed++
		case "FAILURE", "ERROR", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE", "STALE":
			summary.Failed++
		case "SKIPPED", "NEUTRAL":
			summary.Skipped++
		case "PENDING", "EXPECTED", "QUEUED", "IN_PROGRESS", "WAITING", "REQUESTED":
			summary.Pending++
		default:
			// A completed check without a conclusion, or a new provider state we
			// do not understand yet, must never be projected as passed.
			summary.Pending++
		}
	}
	return summary
}

func firstString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := item[key]
		if !ok || value == nil {
			continue
		}
		if text, ok := value.(string); ok && text != "" {
			return text
		}
	}
	return ""
}

func parseGitHubRemote(remote string) (repository, webURL string, ok bool) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", "", false
	}

	if strings.HasPrefix(remote, "git@github.com:") {
		return normalizeGitHubRepository(strings.TrimPrefix(remote, "git@github.com:"))
	}

	parsed, err := url.Parse(remote)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", "", false
	}
	return normalizeGitHubRepository(strings.TrimPrefix(parsed.Path, "/"))
}

func normalizeGitHubRepository(path string) (repository, webURL string, ok bool) {
	path = strings.TrimSuffix(strings.TrimSpace(path), ".git")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	repository = parts[0] + "/" + parts[1]
	return repository, "https://github.com/" + repository, true
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return nil, fmt.Errorf("%s: %w: %s", name, err, detail)
}
