package hoststate

import (
	"testing"
	"time"
)

func TestRepositoryAgentHistoryGroupsSessionsByAgentAndKeepsLastSeen(t *testing.T) {
	now := time.Date(2026, time.August, 23, 2, 0, 0, 0, time.FixedZone("EEST", 3*60*60))
	repo := Repository{Sessions: []Session{
		{Agent: "Claude", Active: false, LastSeenAt: now.Add(-48 * time.Hour)},
		{Agent: "Claude", Active: true, LastSeenAt: now.Add(-30 * time.Second)},
		{Agent: "Codex", Active: false, LastSeenAt: now.Add(-10 * 24 * time.Hour)},
		{Agent: "Cursor", Active: false, LastSeenAt: now.Add(-2 * 24 * time.Hour)},
	}}

	history := repo.AgentHistory()
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3", len(history))
	}

	if history[0].Label != "Claude" || !history[0].Active || !history[0].LastSeenAt.Equal(now.Add(-30*time.Second)) {
		t.Fatalf("first entry = %#v, want active Claude with latest last seen", history[0])
	}
	if history[1].Label != "Cursor" || history[1].Active {
		t.Fatalf("second entry = %#v, want inactive Cursor", history[1])
	}
	if history[2].Label != "Codex" || history[2].Active || !history[2].Last {
		t.Fatalf("last entry = %#v, want inactive Codex marked last", history[2])
	}
	if history[0].Last || history[1].Last {
		t.Fatalf("only final entry should be marked last: %#v", history)
	}
}
