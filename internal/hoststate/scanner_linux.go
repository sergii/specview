//go:build linux

package hoststate

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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
	slog.Debug("Linux Codex scan completed",
		"diagnostics", len(diagnostics),
		"observations", len(observations),
		"duration", time.Since(started),
	)
	return observations, nil
}

func diagnoseCodex() ([]ScanDiagnostic, error) {
	slog.Debug("scanning Linux process table for Codex")
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
		if !looksLikeCodex(command) {
			continue
		}

		diagnostic := ScanDiagnostic{
			PID:     pid,
			Command: command,
			Matched: true,
			Stage:   "matched",
		}
		slog.Debug("Codex process matched", "pid", pid)

		cwd, err := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
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
	slog.Debug("Linux Codex diagnostics completed", "matched_processes", len(diagnostics))
	return diagnostics, nil
}
