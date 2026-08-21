//go:build darwin

package hoststate

import (
	"fmt"
	"os/exec"
	"strings"
)

func (CodexScanner) Scan() ([]Observation, error) {
	output, err := exec.Command("ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	var observations []Observation
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
		if !ok || !looksLikeCodex(parts[1]) {
			continue
		}
		cwd, err := darwinProcessCWD(pid)
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

func darwinProcessCWD(pid int) (string, error) {
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
