package hoststate

import (
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Scanner interface {
	Scan() ([]Observation, error)
}

type CodexScanner struct{}

type ScanDiagnostic struct {
	PID            int
	Command        string
	Matched        bool
	CWD            string
	RepositoryRoot string
	Stage          string
	Error          string
}

func NewCodexScanner() Scanner {
	return CodexScanner{}
}

func DiagnoseCodex() ([]ScanDiagnostic, error) {
	return diagnoseCodex()
}

func looksLikeCodex(command string) bool {
	fields := strings.Fields(strings.ToLower(command))
	if len(fields) == 0 {
		return false
	}

	cleanField := func(field string) string {
		return strings.Trim(field, "\"'")
	}
	isCodexExecutable := func(field string) bool {
		clean := cleanField(field)
		base := filepath.Base(clean)
		return base == "codex" || strings.HasPrefix(base, "codex-")
	}

	if isCodexExecutable(fields[0]) {
		return true
	}

	for _, field := range fields[1:] {
		clean := cleanField(field)
		if strings.Contains(clean, "/@openai/codex/") {
			return true
		}
	}

	wrapper := filepath.Base(cleanField(fields[0]))
	switch wrapper {
	case "node", "nodejs", "bun", "deno", "sh", "bash", "zsh", "dash", "npm", "npx", "pnpm", "yarn":
		for _, field := range fields[1:] {
			if isCodexExecutable(field) {
				return true
			}
		}
	}

	return false
}

func canonicalRepositoryRoot(cwd string) (string, error) {
	slog.Debug("resolving canonical Git repository", "cwd", cwd)

	cmd := exec.Command("git", "-C", cwd, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "worktree ") {
				root := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
				if root != "" {
					root = filepath.Clean(root)
					slog.Debug("canonical Git repository resolved from worktree list", "cwd", cwd, "repository", root)
					return root, nil
				}
			}
		}
	} else {
		slog.Debug("git worktree lookup failed; falling back to rev-parse", "cwd", cwd, "error", err)
	}

	cmd = exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	output, err = cmd.Output()
	if err != nil {
		slog.Debug("Git repository resolution failed", "cwd", cwd, "error", err)
		return "", err
	}
	root := filepath.Clean(strings.TrimSpace(string(output)))
	slog.Debug("canonical Git repository resolved from rev-parse", "cwd", cwd, "repository", root)
	return root, nil
}

func parsePID(value string) (int, bool) {
	pid, err := strconv.Atoi(strings.TrimSpace(value))
	return pid, err == nil && pid > 0
}
