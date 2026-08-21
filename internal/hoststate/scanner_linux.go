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
	slog.Debug("scanning Linux process table for Codex")
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var observations []Observation
	matched := 0
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
		command := strings.ReplaceAll(string(cmdline), "\x00", " ")
		if !looksLikeCodex(command) {
			continue
		}
		matched++
		slog.Debug("Codex process matched", "pid", pid)

		cwd, err := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
		if err != nil {
			slog.Debug("Codex cwd lookup failed", "pid", pid, "error", err)
			continue
		}
		root, err := canonicalRepositoryRoot(cwd)
		if err != nil {
			slog.Debug("Codex Git repository lookup failed", "pid", pid, "cwd", cwd, "error", err)
			continue
		}
		observations = append(observations, Observation{
			Agent:          "Codex",
			PID:            pid,
			RepositoryRoot: root,
		})
		slog.Debug("Codex observation produced", "pid", pid, "cwd", cwd, "repository", root)
	}
	slog.Debug("Linux Codex scan completed",
		"matched_processes", matched,
		"observations", len(observations),
		"duration", time.Since(started),
	)
	return observations, nil
}

func diagnoseCodex() ([]ScanDiagnostic, error) {
	observations, err := (CodexScanner{}).Scan()
	if err != nil {
		return nil, err
	}
	diagnostics := make([]ScanDiagnostic, 0, len(observations))
	for _, observation := range observations {
		diagnostics = append(diagnostics, ScanDiagnostic{
			PID:            observation.PID,
			Matched:        true,
			RepositoryRoot: observation.RepositoryRoot,
			Stage:          "ok",
		})
	}
	return diagnostics, nil
}
