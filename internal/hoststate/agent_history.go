package hoststate

import (
	"sort"
	"strings"
	"time"
)

// AgentHistoryEntry is a repository-scoped projection of observed sessions
// for one execution agent. LastSeenAt is the most recent observation across
// the sessions included in the projection.
type AgentHistoryEntry struct {
	Label      string
	Active     bool
	LastSeenAt time.Time
	Last       bool
}

// AgentHistory returns both active and historical agents observed in a
// repository. This is intentionally not the live hierarchy projection.
func (r Repository) AgentHistory() []AgentHistoryEntry {
	return r.agentHistory(false)
}

// ActiveAgentHistory returns only agents with at least one currently active
// session. The host hierarchy uses this projection so it mirrors the live
// repository execution view: when an agent exits, its row disappears.
func (r Repository) ActiveAgentHistory() []AgentHistoryEntry {
	return r.agentHistory(true)
}

func (r Repository) agentHistory(activeOnly bool) []AgentHistoryEntry {
	byAgent := make(map[string]*AgentHistoryEntry)
	for _, session := range r.Sessions {
		if activeOnly && !session.Active {
			continue
		}
		label := strings.TrimSpace(session.Agent)
		if label == "" {
			continue
		}
		entry := byAgent[label]
		if entry == nil {
			entry = &AgentHistoryEntry{Label: label}
			byAgent[label] = entry
		}
		entry.Active = entry.Active || session.Active
		if entry.LastSeenAt.IsZero() || session.LastSeenAt.After(entry.LastSeenAt) {
			entry.LastSeenAt = session.LastSeenAt
		}
	}

	history := make([]AgentHistoryEntry, 0, len(byAgent))
	for _, entry := range byAgent {
		history = append(history, *entry)
	}
	sort.Slice(history, func(i, j int) bool {
		if history[i].Active != history[j].Active {
			return history[i].Active
		}
		if !history[i].LastSeenAt.Equal(history[j].LastSeenAt) {
			return history[i].LastSeenAt.After(history[j].LastSeenAt)
		}
		return history[i].Label < history[j].Label
	})
	if len(history) > 0 {
		history[len(history)-1].Last = true
	}
	return history
}
