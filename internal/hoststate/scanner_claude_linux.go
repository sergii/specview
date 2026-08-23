//go:build linux

package hoststate

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func diagnoseClaude() ([]ScanDiagnostic, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var diagnostics []ScanDiagnostic
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}
		command := strings.TrimSpace(strings.ReplaceAll(string(cmdline), "\x00", " "))
		if !looksLikeClaude(command) {
			continue
		}

		diagnostic := ScanDiagnostic{PID: pid, Command: command, Matched: true, Stage: "matched"}
		cwd, err := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
		if err != nil {
			diagnostic.Stage = "cwd"
			diagnostic.Error = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		diagnostic.CWD = cwd
		root, err := canonicalRepositoryRoot(cwd)
		if err != nil {
			diagnostic.Stage = "git"
			diagnostic.Error = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		diagnostic.RepositoryRoot = root
		diagnostic.Stage = "ok"
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, nil
}
