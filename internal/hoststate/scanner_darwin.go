//go:build darwin

package hoststate

import (
	"fmt"
	"os/exec"
	"strings"
)

func (CodexScanner) Scan() ([]Observation, error) {
	diagnostics, err := diagnoseCodex()
	if err != nil {
		return nil, err
	}
	var observations []Observation
	for _, diagnostic := range diagnostics {
		if diagnostic.Stage != "ok" {
			continue
		}
		observations = append(observations, Observation{
			Agent:          "Codex",
			PID:            diagnostic.PID,
			RepositoryRoot: diagnostic.RepositoryRoot,
		})
	}
	return observations, nil
}

func diagnoseCodex() ([]ScanDiagnostic, error) {
	output, err := exec.Command("ps", "-axww", "-o", "pid=,command=").Output()
	if err != nil {
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
			continue
		}
		diagnostic.CWD = cwd
		diagnostic.Stage = "cwd"
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
