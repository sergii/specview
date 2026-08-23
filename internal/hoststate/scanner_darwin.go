//go:build darwin

package hoststate

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

func (CodexScanner) Scan() ([]Observation, error) {
	started := time.Now()
	diagnostics, err := diagnoseCodex()
	if err != nil {
		return nil, err
	}
	var observations []Observation
	for _, diagnostic := range diagnostics {
		if diagnostic.Stage != "ok" {
			slog.Debug("Codex process skipped",
				"pid", diagnostic.PID,
				"stage", diagnostic.Stage,
				"error", diagnostic.Error,
			)
			continue
		}
		observation := Observation{
			Agent:          "Codex",
			PID:            diagnostic.PID,
			RepositoryRoot: diagnostic.RepositoryRoot,
		}
		observations = append(observations, observation)
		slog.Debug("Codex observation produced",
			"pid", observation.PID,
			"repository", observation.RepositoryRoot,
		)
	}
	slog.Debug("Darwin Codex scan completed",
		"diagnostics", len(diagnostics),
		"observations", len(observations),
		"duration", time.Since(started),
	)
	return observations, nil
}

func diagnoseCodex() ([]ScanDiagnostic, error) {
	slog.Debug("scanning Darwin process table for Codex")
	output, err := exec.Command("ps", "-axww", "-o", "pid=,command=").Output()
	if err != nil {
		slog.Error("Darwin process table scan failed", "error", err)
		return nil, err
	}
	var diagnostics []ScanDiagnostic
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		pid, ok := parsePID(parts[0])
		if !ok {
			continue
		}
		command := strings.TrimSpace(parts[1])
		if !looksLikeCodex(command) {
			continue
		}

		slog.Debug("Codex process matched", "pid", pid)
		diagnostic := ScanDiagnostic{
			PID:     pid,
			Command: command,
			Matched: true,
			Stage:   "matched",
		}
		cwd, err := darwinProcessCWD(pid)
		if err != nil {
			diagnostic.Stage = "cwd"
			diagnostic.Error = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			slog.Debug("Codex cwd lookup failed", "pid", pid, "error", err)
			continue
		}
		diagnostic.CWD = cwd
		diagnostic.Stage = "cwd"
		slog.Debug("Codex cwd resolved", "pid", pid, "cwd", cwd)

		root, err := canonicalRepositoryRoot(cwd)
		if err != nil {
			diagnostic.Stage = "git"
			diagnostic.Error = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			slog.Debug("Codex Git repository lookup failed", "pid", pid, "cwd", cwd, "error", err)
			continue
		}
		diagnostic.RepositoryRoot = root
		diagnostic.Stage = "ok"
		diagnostics = append(diagnostics, diagnostic)
		slog.Debug("Codex discovery succeeded", "pid", pid, "cwd", cwd, "repository", root)
	}
	slog.Debug("Darwin Codex diagnostics completed", "matched_processes", len(diagnostics))
	return diagnostics, nil
}

func darwinProcessCWD(pid int) (string, error) {
	slog.Debug("reading Darwin process cwd", "pid", pid)
	output, err := exec.Command("lsof", "-a", "-p", fmt.Sprintf("%d", pid), "-d", "cwd", "-Fn").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "n") && len(line) > 1 {
			return strings.TrimPrefix(line, "n"), nil
		}
	}
	return "", fmt.Errorf("cwd not found for pid %d", pid)
}
