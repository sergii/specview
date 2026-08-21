package hoststate

import (
	"path/filepath"
	"sort"
	"strings"
)

// RepositoryExecutionView is a read-only projection of the current execution
// context for one repository. Git remains authoritative for repository and
// worktree state; execution adapters remain authoritative for active sessions.
type RepositoryExecutionView struct {
	Remote    string
	Worktrees []Worktree
	Sessions  []ExecutionSession
	Error     string
}

type Worktree struct {
	Path       string
	Branch     string
	Head       string
	Detached   bool
	DirtyCount int
	Agents     []string
}

func (w Worktree) BranchLabel() string {
	if w.Detached {
		return "detached"
	}
	if w.Branch == "" {
		return "unknown"
	}
	return w.Branch
}

func (w Worktree) ShortHead() string {
	if len(w.Head) <= 8 {
		return w.Head
	}
	return w.Head[:8]
}

func (w Worktree) ChangeLabel() string {
	switch w.DirtyCount {
	case 0:
		return "clean"
	case 1:
		return "1 change"
	default:
		return itoa(w.DirtyCount) + " changes"
	}
}

func (w Worktree) AgentLabel() string {
	if len(w.Agents) == 0 {
		return "idle"
	}
	if len(w.Agents) == 1 {
		return w.Agents[0]
	}
	return itoa(len(w.Agents)) + " agents"
}

// ExecutionView accepts an optional source so the host server can pass the same
// registry used by the runtime. The default preserves the convenient template
// method for callers that do not need dependency injection.
func (r Repository) ExecutionView(sources ...ExecutionSource) RepositoryExecutionView {
	view, err := inspectGitRepository(r.Root)
	if err != nil {
		view.Error = err.Error()
		return view
	}

	var source ExecutionSource = DefaultExecutionRegistry()
	if len(sources) > 0 && sources[0] != nil {
		source = sources[0]
	}

	sessions, err := source.Sessions()
	if err != nil {
		view.Error = err.Error()
		return view
	}
	for _, session := range sessions {
		if filepath.Clean(session.RepositoryRoot) != filepath.Clean(r.Root) {
			continue
		}
		session.StartedAt = r.startedAtForProcesses(session.Agent, session.ProcessIDs)
		view.Sessions = append(view.Sessions, session)
	}
	sort.Slice(view.Sessions, func(i, j int) bool {
		if view.Sessions[i].CWD == view.Sessions[j].CWD {
			return view.Sessions[i].Adapter < view.Sessions[j].Adapter
		}
		return view.Sessions[i].CWD < view.Sessions[j].CWD
	})

	for i := range view.Worktrees {
		seen := make(map[string]struct{})
		for _, session := range view.Sessions {
			if filepath.Clean(session.WorktreeRoot) != filepath.Clean(view.Worktrees[i].Path) {
				continue
			}
			if _, ok := seen[session.Agent]; ok {
				continue
			}
			seen[session.Agent] = struct{}{}
			view.Worktrees[i].Agents = append(view.Worktrees[i].Agents, session.Agent)
		}
		sort.Strings(view.Worktrees[i].Agents)
	}
	return view
}

func (r Repository) startedAtForProcesses(agent string, processIDs []int) (startedAt time.Time) {
	pidSet := make(map[int]struct{}, len(processIDs))
	for _, pid := range processIDs {
		pidSet[pid] = struct{}{}
	}
	for _, persisted := range r.Sessions {
		if !persisted.Active || persisted.Agent != agent {
			continue
		}
		if _, ok := pidSet[persisted.PID]; !ok {
			continue
		}
		if startedAt.IsZero() || persisted.StartedAt.Before(startedAt) {
			startedAt = persisted.StartedAt
		}
	}
	return startedAt
}

func inspectGitRepository(root string) (RepositoryExecutionView, error) {
	view := RepositoryExecutionView{}

	if output, err := runGit(root, "remote", "get-url", "origin"); err == nil {
		view.Remote = strings.TrimSpace(string(output))
	}

	output, err := runGit(root, "worktree", "list", "--porcelain")
	if err != nil {
		return view, err
	}
	view.Worktrees = parseWorktrees(string(output))
	for i := range view.Worktrees {
		status, statusErr := runGit(view.Worktrees[i].Path, "status", "--porcelain")
		if statusErr != nil {
			continue
		}
		view.Worktrees[i].DirtyCount = countNonEmptyLines(string(status))
	}
	return view, nil
}

func parseWorktrees(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree
	flush := func() {
		if current == nil || current.Path == "" {
			return
		}
		worktrees = append(worktrees, *current)
		current = nil
	}

	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			flush()
			current = &Worktree{Path: filepath.Clean(strings.TrimSpace(strings.TrimPrefix(line, "worktree ")))}
			continue
		}
		if current == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "branch ")), "refs/heads/")
		case line == "detached":
			current.Detached = true
		}
	}
	flush()
	return worktrees
}

func countNonEmptyLines(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
