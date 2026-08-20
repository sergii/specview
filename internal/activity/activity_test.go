package activity

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActiveForReturnsOnlyFreshWorkingSessions(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 20, 18, 0, 30, 0, time.UTC)

	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("fresh.json", `{
  "version": 1,
  "session_id": "fresh",
  "agent": {"id": "codex", "label": "Codex"},
  "spec": "specs/H10-thin-editorial-kanban.md",
  "state": "working",
  "started_at": "2026-08-20T18:00:00Z",
  "heartbeat_at": "2026-08-20T18:00:20Z"
}`)
	write("stale.json", `{
  "version": 1,
  "session_id": "stale",
  "agent": {"id": "claude-code", "label": "Claude Code"},
  "spec": "specs/H10-thin-editorial-kanban.md",
  "state": "working",
  "heartbeat_at": "2026-08-20T17:59:00Z"
}`)
	write("other.json", `{
  "version": 1,
  "session_id": "other",
  "agent": {"id": "opencode", "label": "OpenCode"},
  "spec": "specs/H11-scale-and-performance-observability.md",
  "state": "working",
  "heartbeat_at": "2026-08-20T18:00:20Z"
}`)

	store := NewStore(root)
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}

	active := store.ActiveFor("specs/H10-thin-editorial-kanban.md", now)
	if len(active) != 1 {
		t.Fatalf("active sessions = %d, want 1", len(active))
	}
	if got := AgentLabel(active[0]); got != "Codex" {
		t.Fatalf("agent label = %q, want Codex", got)
	}
}

func TestInvalidActivityRecordDoesNotBreakStore(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte(`{"version":`), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(root)
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	if len(store.Errors()) != 1 {
		t.Fatalf("parse errors = %d, want 1", len(store.Errors()))
	}
}

func TestAgentLabelFallsBackSafely(t *testing.T) {
	if got := AgentLabel(Record{Agent: Agent{ID: "codex"}}); got != "codex" {
		t.Fatalf("label = %q, want codex", got)
	}
	if got := AgentLabel(Record{}); got != "Agent" {
		t.Fatalf("label = %q, want Agent", got)
	}
}
