package controlplane

const SchemaVersion = 1

type ListRepositoriesResult struct {
	SchemaVersion int                 `json:"schema_version"`
	Host          string              `json:"host"`
	Repositories  []RepositorySummary `json:"repositories"`
	Warnings      []string            `json:"warnings,omitempty"`
}

type RepositorySummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Root         string   `json:"root"`
	Active       bool     `json:"active"`
	Agents       []string `json:"agents,omitempty"`
	FirstSeenAt  string   `json:"first_seen_at,omitempty"`
	LastSeenAt   string   `json:"last_seen_at,omitempty"`
	SpecAdapter  string   `json:"spec_adapter,omitempty"`
	SpecLabel    string   `json:"spec_label,omitempty"`
	SpecDetected bool     `json:"spec_detected"`
	SpecSupported bool    `json:"spec_supported"`
}

type GetRepositoryResult struct {
	SchemaVersion int              `json:"schema_version"`
	Host          string           `json:"host"`
	Repository    RepositoryDetail `json:"repository"`
	Warnings      []string         `json:"warnings,omitempty"`
}

type RepositoryDetail struct {
	RepositorySummary
	Git   *GitSummary   `json:"git,omitempty"`
	Forge *ForgeSummary `json:"forge,omitempty"`
}

type GitSummary struct {
	Remote    string            `json:"remote,omitempty"`
	Worktrees []WorktreeSummary `json:"worktrees"`
}

type ForgeSummary struct {
	Provider   string               `json:"provider"`
	Matched    bool                 `json:"matched"`
	Available  bool                 `json:"available"`
	Repository string               `json:"repository,omitempty"`
	WebURL     string               `json:"web_url,omitempty"`
	Error      string               `json:"error,omitempty"`
	PullRequests []PullRequestSummary `json:"pull_requests,omitempty"`
}

type PullRequestSummary struct {
	Number  int          `json:"number"`
	Title   string       `json:"title"`
	URL     string       `json:"url"`
	State   string       `json:"state"`
	Draft   bool         `json:"draft"`
	Base    string       `json:"base"`
	Head    string       `json:"head"`
	Checks  CheckSummary `json:"checks"`
}

type CheckSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
	Skipped int `json:"skipped"`
}

type ListActiveSessionsResult struct {
	SchemaVersion int              `json:"schema_version"`
	Host          string           `json:"host"`
	Sessions      []SessionSummary `json:"sessions"`
	Warnings      []string         `json:"warnings,omitempty"`
}

type SessionSummary struct {
	ID             string `json:"id"`
	Adapter        string `json:"adapter"`
	Agent          string `json:"agent"`
	RepositoryID   string `json:"repository_id"`
	RepositoryRoot string `json:"repository_root"`
	WorktreeRoot   string `json:"worktree_root,omitempty"`
	CWD            string `json:"cwd"`
	ProcessIDs     []int  `json:"process_ids,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
}

type ListWorktreesResult struct {
	SchemaVersion int               `json:"schema_version"`
	Host          string            `json:"host"`
	RepositoryID  string            `json:"repository_id"`
	RepositoryName string           `json:"repository_name"`
	Worktrees     []WorktreeSummary `json:"worktrees"`
	Warnings      []string          `json:"warnings,omitempty"`
}

type WorktreeSummary struct {
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
