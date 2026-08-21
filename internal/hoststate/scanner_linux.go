//go:build linux

package hoststate

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (CodexScanner) Scan() ([]Observation, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var observations []Observation
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
		cwd, err := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
		if err != nil {
			continue
		}
		root, err := canonicalRepositoryRoot(cwd)
		if err != nil {
			continue
		}
		observations = append(observations, Observation{
			Agent:          "Codex",
			PID:            pid,
			RepositoryRoot: root,
		})
	}
	return observations, nil
}
