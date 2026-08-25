package mcpserver

import (
	"fmt"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationruntime"
)

const federationHostResultSchemaVersion = 1

type federationHostRepositoryResult struct {
	GroupID  string                     `json:"group_id"`
	Name     string                     `json:"name"`
	Active   bool                       `json:"active"`
	Agents   []string                   `json:"agents,omitempty"`
	Instance federation.SourcedInstance `json:"instance"`
}

type federationHostResult struct {
	SchemaVersion int                              `json:"schema_version"`
	GeneratedAt   time.Time                        `json:"generated_at"`
	Host          federationruntime.HostStatus     `json:"host"`
	Repositories  []federationHostRepositoryResult `json:"repositories"`
}

func projectFederationHost(projection federationruntime.Projection, hostID string) (federationHostResult, error) {
	selected, found := federationruntime.SelectHost(projection, hostID)
	if !found {
		return federationHostResult{}, fmt.Errorf("federation Host %q not found", hostID)
	}

	selections := federationruntime.RepositoriesForHost(projection, hostID)
	result := federationHostResult{
		SchemaVersion: federationHostResultSchemaVersion,
		GeneratedAt:   projection.GeneratedAt,
		Host:          selected,
		Repositories:  make([]federationHostRepositoryResult, 0, len(selections)),
	}
	for _, selection := range selections {
		result.Repositories = append(result.Repositories, federationHostRepositoryResult{
			GroupID:  selection.Group.GroupID,
			Name:     selection.Group.Name,
			Active:   selection.Group.Active,
			Agents:   append([]string(nil), selection.Group.Agents...),
			Instance: selection.Instance,
		})
	}
	return result, nil
}
