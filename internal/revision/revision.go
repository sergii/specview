package revision

import (
	"path/filepath"
	"strings"

	"github.com/sergii/specview/internal/sourcecontrol"
)

const (
	ReasonWorktreeNotFound = "worktree_not_found"
	ReasonDirtyWorktree    = "dirty_worktree"
	ReasonHeadUnavailable  = "head_unavailable"
)

type Resolution struct {
	Revision     string
	WorktreePath string
	Available    bool
	Reason       string
}

func ResolveGit(projectRoot string, git sourcecontrol.GitContext) Resolution {
	worktree, ok := containingWorktree(projectRoot, git.Worktrees)
	if !ok {
		return Resolution{Reason: ReasonWorktreeNotFound}
	}

	resolution := Resolution{WorktreePath: worktree.Path}
	if worktree.DirtyCount > 0 {
		resolution.Reason = ReasonDirtyWorktree
		return resolution
	}
	if strings.TrimSpace(worktree.Head) == "" {
		resolution.Reason = ReasonHeadUnavailable
		return resolution
	}

	resolution.Revision = "git:" + worktree.Head
	resolution.Available = true
	return resolution
}

func containingWorktree(projectRoot string, worktrees []sourcecontrol.Worktree) (sourcecontrol.Worktree, bool) {
	root := filepath.Clean(projectRoot)
	var best sourcecontrol.Worktree
	bestLength := -1

	for _, worktree := range worktrees {
		path := filepath.Clean(worktree.Path)
		if !contains(path, root) {
			continue
		}
		if len(path) > bestLength {
			best = worktree
			bestLength = len(path)
		}
	}
	return best, bestLength >= 0
}

func contains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
