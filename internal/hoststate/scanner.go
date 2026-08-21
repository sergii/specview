package hoststate

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Scanner interface {
	Scan() ([]Observation, error)
}

type CodexScanner struct{}

func NewCodexScanner() Scanner {
	return CodexScanner{}
}

func looksLikeCodex(command string) bool {
	fields := strings.Fields(strings.ToLower(command))
	if len(fields) == 0 {
		return false
	}
	executable := strings.Trim(fields[0], "\"'")
	base := filepath.Base(executable)
	if base == "codex" || strings.HasPrefix(base, "codex-") {
		return true
	}
	for _, field := range fields[1:] {
		clean := strings.Trim(field, "\"'")
		if strings.Contains(clean, "/@openai/codex/") {
			return true
		}
	}
	return false
}

func canonicalRepositoryRoot(cwd string) (string, error) {
	cmd := exec.Command("git", "-C", cwd, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "worktree ") {
				root := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
				if root != "" {
					return filepath.Clean(root), nil
				}
			}
		}
	}

	cmd = exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	output, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(string(output))), nil
}

func parsePID(value string) (int, bool) {
	pid, err := strconv.Atoi(strings.TrimSpace(value))
	return pid, err == nil && pid > 0
}
