package hoststate

import (
	"fmt"
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

func DiagnoseClaude() ([]ScanDiagnostic, error) {
	return diagnoseClaude()
}

// normalizeFilesystemPath returns a stable path identity while preserving
// callers' original path spelling for display. This matters on systems such as
// macOS where /var and /private/var can refer to the same directory.
func normalizeFilesystemPath(path string) string {
	clean := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(clean)
	if err == nil && resolved != "" {
		return filepath.Clean(resolved)
	}
	return clean
}

func sameFilesystemPath(left, right string) bool {
	return normalizeFilesystemPath(left) == normalizeFilesystemPath(right)
}

func looksLikeCodex(command string) bool {
	return looksLikeAgentCommand(command, "codex", "/@openai/codex/")
}

func looksLikeClaude(command string) bool {
	return looksLikeAgentCommand(command, "claude", "/@anthropic-ai/claude-code/")
}

func looksLikeAgentCommand(command, executable, packagePath string) bool {
	fields := strings.Fields(strings.ToLower(command))
	if len(fields) == 0 {
		return false
	}

	cleanField := func(field string) string {
		return strings.Trim(field, "\"'")
	}
	isAgentExecutable := func(field string) bool {
		clean := cleanField(field)
		base := filepath.Base(clean)
		return base == executable || strings.HasPrefix(base, executable+"-")
	}

	if isAgentExecutable(fields[0]) {
		return true
	}

	for _, field := range fields[1:] {
		clean := cleanField(field)
		if packagePath != "" && strings.Contains(clean, packagePath) {
			return true
		}
	}

	wrapper := filepath.Base(cleanField(fields[0]))
	switch wrapper {
	case "node", "nodejs", "bun", "deno", "sh", "bash", "zsh", "dash", "npm", "npx", "pnpm", "yarn":
		for _, field := range fields[1:] {
			if isAgentExecutable(field) {
				return true
			}
		}
	}

	return false
}

func canonicalRepositoryRoot(cwd string) (string, error) {
	slog.Debug("resolving canonical Git repository", "cwd", cwd)

	output, err := runGit(cwd, "worktree", "list", "--porcelain")
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

	output, err = runGit(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		slog.Debug("Git repository resolution failed", "cwd", cwd, "error", err)
		return "", err
	}
	root := filepath.Clean(strings.TrimSpace(string(output)))
	slog.Debug("canonical Git repository resolved from rev-parse", "cwd", cwd, "repository", root)
	return root, nil
}

func runGit(cwd string, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", cwd}, args...)
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

func parsePID(value string) (int, bool) {
	pid, err := strconv.Atoi(strings.TrimSpace(value))
	return pid, err == nil && pid > 0
}
