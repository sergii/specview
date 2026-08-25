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
	var selected federationruntime.HostStatus
	found := false
	for _, host := range projection.Hosts {
		if host.HostID != hostID {
			continue
		}
		selected = host
		found = true
		break
	}
	if !found {
		return federationHostResult{}, fmt.Errorf("federation Host %q not found", hostID)
	}

	result := federationHostResult{
		SchemaVersion: federationHostResultSchemaVersion,
		GeneratedAt:   projection.GeneratedAt,
		Host:          selected,
		Repositories:  make([]federationHostRepositoryResult, 0),
	}
	for _, group := range projection.Federation.Repositories {
		for _, instance := range group.Instances {
			if instance.HostID != hostID {
				continue
			}
			result.Repositories = append(result.Repositories, federationHostRepositoryResult{
				GroupID:  group.GroupID,
				Name:     group.Name,
				Active:   group.Active,
				Agents:   append([]string(nil), group.Agents...),
				Instance: instance,
			})
		}
	}
	return result, nil
}
