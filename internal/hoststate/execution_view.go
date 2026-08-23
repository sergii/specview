package hoststate

import (
	"sort"
	"time"

	"github.com/sergii/specview/internal/sourcecontrol"
)

// RepositoryExecutionView is a read-only projection of current execution
// context for one repository. Source-control mechanics are provided by the
// portable sourcecontrol layer; execution adapters remain authoritative for
// active sessions.
type RepositoryExecutionView struct {
	Remote    string
	Worktrees []Worktree
	Sessions  []ExecutionSession
	Error     string
}

type Worktree struct {
	sourcecontrol.Worktree
	Agents []string
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

func (w Worktree) DisplayPath() string {
	return repositoryDisplayName(w.Path)
}

func (s ExecutionSession) DisplayCWD() string {
	return repositoryDisplayName(s.CWD)
}

func (s ExecutionSession) DisplayWorktreeRoot() string {
	return repositoryDisplayName(s.WorktreeRoot)
}

// ExecutionView keeps the H13 convenience API while delegating local Git
// inspection to the source-control layer introduced by H15.
func (r Repository) ExecutionView(sources ...ExecutionSource) RepositoryExecutionView {
	gitContext, err := sourcecontrol.InspectGit(r.Root)
	if err != nil {
		return RepositoryExecutionView{Error: err.Error()}
	}
	return r.ExecutionViewWithGit(gitContext, sources...)
}

// ExecutionViewWithGit allows the web projection to reuse the exact Git
// snapshot already collected for RepositoryContext rather than issuing a
// duplicate set of Git commands.
func (r Repository) ExecutionViewWithGit(gitContext sourcecontrol.GitContext, sources ...ExecutionSource) RepositoryExecutionView {
	view := RepositoryExecutionView{Remote: gitContext.Remote}
	for _, worktree := range gitContext.Worktrees {
		view.Worktrees = append(view.Worktrees, Worktree{Worktree: worktree})
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
		if !sameFilesystemPath(session.RepositoryRoot, r.Root) {
			continue
		}
		session.StartedAt = r.startedAtForExecution(session)
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
			if !sameFilesystemPath(session.WorktreeRoot, view.Worktrees[i].Path) {
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

func (r Repository) startedAtForExecution(live ExecutionSession) (startedAt time.Time) {
	startedAt = live.StartedAt
	for _, persisted := range r.Sessions {
		if !persisted.Active {
			continue
		}
		if persisted.IdentityKind == SessionIdentityLogical && persisted.ID == live.ID {
			return earlierNonZero(startedAt, persisted.StartedAt)
		}
	}

	// During a v1->v2 cutover the runtime may render a live logical session
	// before its migrated PID fragments have been collapsed and persisted.
	for _, persisted := range r.Sessions {
		if !legacyFragmentMatches(persisted, live) {
			continue
		}
		startedAt = earlierNonZero(startedAt, persisted.StartedAt)
	}
	return startedAt
}

func earlierNonZero(left, right time.Time) time.Time {
	if left.IsZero() {
		return right
	}
	if right.IsZero() || left.Before(right) {
		return left
	}
	return right
}

// Compatibility helpers keep H13 tests focused on the same observable contract
// while Git parsing now lives in internal/sourcecontrol.
func inspectGitRepository(root string) (RepositoryExecutionView, error) {
	gitContext, err := sourcecontrol.InspectGit(root)
	if err != nil {
		return RepositoryExecutionView{}, err
	}
	view := RepositoryExecutionView{Remote: gitContext.Remote}
	for _, worktree := range gitContext.Worktrees {
		view.Worktrees = append(view.Worktrees, Worktree{Worktree: worktree})
	}
	return view, nil
}

func parseWorktrees(output string) []Worktree {
	parsed := sourcecontrol.ParseWorktrees(output)
	worktrees := make([]Worktree, 0, len(parsed))
	for _, worktree := range parsed {
		worktrees = append(worktrees, Worktree{Worktree: worktree})
	}
	return worktrees
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
