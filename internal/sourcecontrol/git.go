package sourcecontrol

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func InspectGit(root string) (GitContext, error) {
	context := GitContext{}
	if output, err := runGit(root, "remote", "get-url", "origin"); err == nil {
		context.Remote = strings.TrimSpace(string(output))
	}

	output, err := runGit(root, "worktree", "list", "--porcelain")
	if err != nil {
		return GitContext{}, err
	}
	context.Worktrees = ParseWorktrees(string(output))
	for i := range context.Worktrees {
		enrichWorktree(&context.Worktrees[i])
	}
	return context, nil
}

func ParseWorktrees(output string) []Worktree {
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

func enrichWorktree(worktree *Worktree) {
	if output, err := runGit(worktree.Path, "status", "--porcelain"); err == nil {
		worktree.DirtyCount = countNonEmptyLines(string(output))
	}
	if output, err := runGit(worktree.Path, "log", "-1", "--format=%s"); err == nil {
		worktree.LastCommit = strings.TrimSpace(string(output))
	}
	if worktree.Detached {
		return
	}

	upstreamOutput, err := runGit(worktree.Path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return
	}
	worktree.Upstream = strings.TrimSpace(string(upstreamOutput))
	if worktree.Upstream == "" {
		return
	}

	countsOutput, err := runGit(worktree.Path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return
	}
	fields := strings.Fields(string(countsOutput))
	if len(fields) != 2 {
		return
	}
	ahead, aheadErr := strconv.Atoi(fields[0])
	behind, behindErr := strconv.Atoi(fields[1])
	if aheadErr == nil {
		worktree.Ahead = ahead
	}
	if behindErr == nil {
		worktree.Behind = behind
	}
}

func runGit(root string, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", commandArgs...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
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
