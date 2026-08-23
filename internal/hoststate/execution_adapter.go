package hoststate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ExecutionSession is the normalized logical execution unit produced by an
// execution adapter. Process IDs are runtime details, not session identity.
type ExecutionSession struct {
	Adapter        string
	ID             string
	Agent          string
	CWD            string
	RepositoryRoot string
	WorktreeRoot   string
	ProcessIDs     []int
	StartedAt      time.Time
}

func (s ExecutionSession) ProcessLabel() string {
	switch len(s.ProcessIDs) {
	case 0:
		return ""
	case 1:
		return "PID " + itoa(s.ProcessIDs[0])
	default:
		return itoa(len(s.ProcessIDs)) + " processes"
	}
}

// ExecutionAdapter observes one concrete execution environment and normalizes
// its runtime-specific process model into logical execution sessions.
type ExecutionAdapter interface {
	Name() string
	Sessions() ([]ExecutionSession, error)
	Diagnostics() ([]ScanDiagnostic, error)
}

// ExecutionSource is consumed by projections that only need normalized sessions.
type ExecutionSource interface {
	Sessions() ([]ExecutionSession, error)
}

// ExecutionRegistry aggregates independent execution adapters. It also
// implements Scanner so the existing host catalog can consume normalized
// sessions without knowing which agent produced them.
type ExecutionRegistry struct {
	adapters []ExecutionAdapter
}

func NewExecutionRegistry(adapters ...ExecutionAdapter) *ExecutionRegistry {
	copyAdapters := append([]ExecutionAdapter(nil), adapters...)
	return &ExecutionRegistry{adapters: copyAdapters}
}

func DefaultExecutionRegistry() *ExecutionRegistry {
	return NewExecutionRegistry(CodexExecutionAdapter{}, ClaudeExecutionAdapter{})
}

func (r *ExecutionRegistry) Adapter(name string) (ExecutionAdapter, bool) {
	for _, adapter := range r.adapters {
		if adapter.Name() == name {
			return adapter, true
		}
	}
	return nil, false
}

func (r *ExecutionRegistry) Sessions() ([]ExecutionSession, error) {
	var sessions []ExecutionSession
	var failures []error

	for _, adapter := range r.adapters {
		observed, err := adapter.Sessions()
		if err != nil {
			wrapped := fmt.Errorf("%s execution adapter: %w", adapter.Name(), err)
			failures = append(failures, wrapped)
			slog.Warn("execution adapter failed", "adapter", adapter.Name(), "error", err)
			continue
		}
		sessions = append(sessions, observed...)
		slog.Debug("execution adapter observed sessions", "adapter", adapter.Name(), "sessions", len(observed))
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].RepositoryRoot != sessions[j].RepositoryRoot {
			return sessions[i].RepositoryRoot < sessions[j].RepositoryRoot
		}
		if sessions[i].CWD != sessions[j].CWD {
			return sessions[i].CWD < sessions[j].CWD
		}
		return sessions[i].Adapter < sessions[j].Adapter
	})

	if len(sessions) == 0 && len(failures) > 0 {
		return nil, errors.Join(failures...)
	}
	return sessions, nil
}

func (r *ExecutionRegistry) Scan() ([]Observation, error) {
	sessions, err := r.Sessions()
	if err != nil {
		return nil, err
	}

	var observations []Observation
	for _, session := range sessions {
		for _, pid := range session.ProcessIDs {
			observations = append(observations, Observation{
				Agent:          session.Agent,
				PID:            pid,
				RepositoryRoot: session.RepositoryRoot,
			})
		}
	}
	return observations, nil
}

// CodexExecutionAdapter observes Codex. Platform mechanics remain below the
// adapter boundary in the OS-specific scanner files.
type CodexExecutionAdapter struct{}

func (CodexExecutionAdapter) Name() string { return "codex" }

func (CodexExecutionAdapter) Diagnostics() ([]ScanDiagnostic, error) {
	return DiagnoseCodex()
}

func (CodexExecutionAdapter) Sessions() ([]ExecutionSession, error) {
	diagnostics, err := DiagnoseCodex()
	if err != nil {
		return nil, err
	}
	return sessionsFromDiagnostics("codex", "Codex", diagnostics), nil
}

// ClaudeExecutionAdapter observes Claude Code using the same normalized
// execution contract as Codex.
type ClaudeExecutionAdapter struct{}

func (ClaudeExecutionAdapter) Name() string { return "claude" }

func (ClaudeExecutionAdapter) Diagnostics() ([]ScanDiagnostic, error) {
	return DiagnoseClaude()
}

func (ClaudeExecutionAdapter) Sessions() ([]ExecutionSession, error) {
	diagnostics, err := DiagnoseClaude()
	if err != nil {
		return nil, err
	}
	return sessionsFromDiagnostics("claude", "Claude", diagnostics), nil
}

func sessionsFromDiagnostics(adapter, agent string, diagnostics []ScanDiagnostic) []ExecutionSession {
	byKey := make(map[string]*ExecutionSession)

	for _, diagnostic := range diagnostics {
		if diagnostic.Stage != "ok" || diagnostic.RepositoryRoot == "" || diagnostic.CWD == "" {
			continue
		}

		repositoryRoot := filepath.Clean(diagnostic.RepositoryRoot)
		cwd := filepath.Clean(diagnostic.CWD)
		worktreeRoot := cwd
		if output, err := runGit(cwd, "rev-parse", "--show-toplevel"); err == nil {
			candidate := filepath.Clean(strings.TrimSpace(string(output)))
			if !sameFilesystemPath(candidate, cwd) {
				worktreeRoot = candidate
			}
		}

		identityRepository := normalizeFilesystemPath(repositoryRoot)
		identityCWD := normalizeFilesystemPath(cwd)
		key := adapter + "\x00" + identityRepository + "\x00" + identityCWD
		session := byKey[key]
		if session == nil {
			session = &ExecutionSession{
				Adapter:        adapter,
				ID:             executionSessionID(adapter, repositoryRoot, cwd),
				Agent:          agent,
				CWD:            cwd,
				RepositoryRoot: repositoryRoot,
				WorktreeRoot:   worktreeRoot,
			}
			byKey[key] = session
		}
		session.ProcessIDs = append(session.ProcessIDs, diagnostic.PID)
	}

	sessions := make([]ExecutionSession, 0, len(byKey))
	for _, session := range byKey {
		sort.Ints(session.ProcessIDs)
		sessions = append(sessions, *session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].RepositoryRoot != sessions[j].RepositoryRoot {
			return sessions[i].RepositoryRoot < sessions[j].RepositoryRoot
		}
		if sessions[i].CWD != sessions[j].CWD {
			return sessions[i].CWD < sessions[j].CWD
		}
		return sessions[i].Adapter < sessions[j].Adapter
	})
	return sessions
}

func executionSessionID(adapter, repositoryRoot, cwd string) string {
	sum := sha256.Sum256([]byte(adapter + "\x00" + normalizeFilesystemPath(repositoryRoot) + "\x00" + normalizeFilesystemPath(cwd)))
	return "execution-" + hex.EncodeToString(sum[:8])
}
