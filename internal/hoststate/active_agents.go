package hoststate

import "sort"

// ActiveAgents returns the unique active agent labels for this repository.
// Process/helper multiplicity is intentionally collapsed at the projection
// boundary so host views render logical agents rather than OS processes.
func (r Repository) ActiveAgents() []string {
	seen := make(map[string]struct{})
	agents := make([]string, 0)
	for _, session := range r.Sessions {
		if !session.Active || session.Agent == "" {
			continue
		}
		if _, ok := seen[session.Agent]; ok {
			continue
		}
		seen[session.Agent] = struct{}{}
		agents = append(agents, session.Agent)
	}
	sort.Strings(agents)
	return agents
}
