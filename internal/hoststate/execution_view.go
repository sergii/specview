package hoststate

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RepositoryExecutionView is a read-only projection of the current execution
// context for one repository. Git remains authoritative for repository and
// worktree state; process discovery remains authoritative for active sessions.
type RepositoryExecutionView struct {
	Remote    string
	Worktrees []Worktree
	Sessions  []ExecutionContext
	Error     string
}

type ExecutionContext struct {
	Agent        string
	CWD          string
	WorktreeRoot string
	ProcessIDs   []int
	StartedAt    time.Time
}

func (c ExecutionContext) ProcessLabel() string {
	switch len(c.ProcessIDs) {
	case 0:
		return ""
	case 1:
		return "PID " + itoa(c.ProcessIDs[0])
	default:
		return itoa(len(c.ProcessIDs)) + " processes"
	}
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

func (r Repository) ExecutionView() RepositoryExecutionView {
	view, err := inspectGitRepository(r.Root)
	if err != nil {
		view.Error = err.Error()
		return view
	}

	view.Sessions = currentCodexExecution(r)
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

func currentCodexExecution(repo Repository) []ExecutionContext {
	diagnostics, err := DiagnoseCodex()
	if err != nil {
		return nil
	}

	byKey := make(map[string]*ExecutionContext)
	for _, diagnostic := range diagnostics {
		if diagnostic.Stage != "ok" || filepath.Clean(diagnostic.RepositoryRoot) != filepath.Clean(repo.Root) {
			continue
		}
		worktreeRoot := diagnostic.CWD
		if output, gitErr := runGit(diagnostic.CWD, "rev-parse", "--show-toplevel"); gitErr == nil {
			worktreeRoot = filepath.Clean(strings.TrimSpace(string(output)))
		}
		key := "Codex\x00" + filepath.Clean(diagnostic.CWD)
		context := byKey[key]
		if context == nil {
			context = &ExecutionContext{
				Agent:        "Codex",
				CWD:          filepath.Clean(diagnostic.CWD),
				WorktreeRoot: worktreeRoot,
			}
			byKey[key] = context
		}
		context.ProcessIDs = append(context.ProcessIDs, diagnostic.PID)
		for _, persisted := range repo.Sessions {
			if !persisted.Active || persisted.Agent != "Codex" || persisted.PID != diagnostic.PID {
				continue
			}
			if context.StartedAt.IsZero() || persisted.StartedAt.Before(context.StartedAt) {
				context.StartedAt = persisted.StartedAt
			}
		}
	}

	contexts := make([]ExecutionContext, 0, len(byKey))
	for _, context := range byKey {
		sort.Ints(context.ProcessIDs)
		contexts = append(contexts, *context)
	}
	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].CWD == contexts[j].CWD {
			return contexts[i].Agent < contexts[j].Agent
		}
		return contexts[i].CWD < contexts[j].CWD
	})
	return contexts
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
