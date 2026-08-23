package sourcecontrol

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGitHubRemoteParsing(t *testing.T) {
	for _, remote := range []string{
		"git@github.com:sergii/specview.git",
		"https://github.com/sergii/specview.git",
		"ssh://git@github.com/sergii/specview.git",
	} {
		repository, webURL, ok := parseGitHubRemote(remote)
		if !ok || repository != "sergii/specview" || webURL != "https://github.com/sergii/specview" {
			t.Fatalf("parse %q = %q %q %t", remote, repository, webURL, ok)
		}
	}
	if _, _, ok := parseGitHubRemote("git@gitlab.com:sergii/specview.git"); ok {
		t.Fatal("GitLab remote must not match GitHub adapter")
	}
}

func TestGitHubCLIAdapterProjectsObservedBranchPRAndChecks(t *testing.T) {
	adapter := &GitHubCLIAdapter{
		lookPath: func(string) (string, error) { return "/usr/local/bin/gh", nil },
		run: func(context.Context, string, ...string) ([]byte, error) {
			return []byte(`[
  {
    "number": 42,
    "title": "Add execution context",
    "url": "https://github.com/sergii/specview/pull/42",
    "state": "OPEN",
    "isDraft": false,
    "baseRefName": "main",
    "headRefName": "poc/vertical-slice",
    "statusCheckRollup": [
      {"conclusion":"SUCCESS","status":"COMPLETED"},
      {"conclusion":"FAILURE","status":"COMPLETED"},
      {"status":"IN_PROGRESS"},
      {"conclusion":"SKIPPED","status":"COMPLETED"}
    ]
  },
  {
    "number": 99,
    "title": "Unrelated branch",
    "url": "https://github.com/sergii/specview/pull/99",
    "state": "OPEN",
    "isDraft": true,
    "baseRefName": "main",
    "headRefName": "other",
    "statusCheckRollup": []
  }
]`), nil
		},
	}
	context := adapter.Inspect(context.Background(), GitContext{
		Remote: "git@github.com:sergii/specview.git",
		Worktrees: []Worktree{{
			Branch: "poc/vertical-slice",
		}},
	})
	if !context.Matched || !context.Available || context.Error != "" {
		t.Fatalf("provider context = %#v", context)
	}
	if len(context.PullRequests) != 1 {
		t.Fatalf("pull requests = %#v", context.PullRequests)
	}
	pullRequest := context.PullRequests[0]
	if pullRequest.Number != 42 || pullRequest.Checks.Total != 4 || pullRequest.Checks.Passed != 1 || pullRequest.Checks.Failed != 1 || pullRequest.Checks.Pending != 1 || pullRequest.Checks.Skipped != 1 {
		t.Fatalf("pull request = %#v", pullRequest)
	}
	if pullRequest.Checks.Label() != "1 failed · 1 pending · 1 passed · 1 skipped" {
		t.Fatalf("checks label = %q", pullRequest.Checks.Label())
	}
}

func TestGitHubCLIAdapterDegradesWhenGHIsUnavailable(t *testing.T) {
	adapter := &GitHubCLIAdapter{
		lookPath: func(string) (string, error) { return "", errors.New("missing") },
		run:      runCommand,
	}
	context := adapter.Inspect(context.Background(), GitContext{Remote: "https://github.com/sergii/specview.git"})
	if !context.Matched || context.Available || context.Error != "GitHub CLI (gh) is not installed" {
		t.Fatalf("provider context = %#v", context)
	}
}

type fakeProvider struct {
	calls int
}

func (f *fakeProvider) Name() string                { return "Fake" }
func (f *fakeProvider) Supports(remote string) bool { return remote != "" }
func (f *fakeProvider) Inspect(context.Context, GitContext) ProviderContext {
	f.calls++
	return ProviderContext{Name: "Fake", Matched: true, Available: true}
}

func TestServiceCachesRemoteProviderContext(t *testing.T) {
	root := t.TempDir()
	runTestGit(t, root, "init")
	runTestGit(t, root, "remote", "add", "origin", "https://example.invalid/repo.git")

	provider := &fakeProvider{}
	service := NewServiceWithProviderTTL(time.Minute, provider)
	if _, err := service.Inspect(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Inspect(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d", provider.calls)
	}
}
