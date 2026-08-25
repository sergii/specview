package mcpserver

import (
	"fmt"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationruntime"
)

const federationRepositoryResultSchemaVersion = 1

type federationRepositoryGroupResult struct {
	GroupID string   `json:"group_id"`
	Name    string   `json:"name"`
	Active  bool     `json:"active"`
	Agents  []string `json:"agents,omitempty"`
}

type federationRepositoryResult struct {
	SchemaVersion int                             `json:"schema_version"`
	GeneratedAt   time.Time                       `json:"generated_at"`
	Host          federationruntime.HostStatus    `json:"host"`
	Group         federationRepositoryGroupResult `json:"group"`
	Instance      federation.SourcedInstance      `json:"instance"`
}

func projectFederationRepository(projection federationruntime.Projection, hostID, instanceID string) (federationRepositoryResult, error) {
	host, selection, found := federationruntime.SelectRepository(projection, hostID, instanceID)
	if !found {
		return federationRepositoryResult{}, fmt.Errorf("federation repository host %q instance %q not found", hostID, instanceID)
	}
	return federationRepositoryResult{
		SchemaVersion: federationRepositoryResultSchemaVersion,
		GeneratedAt:   projection.GeneratedAt,
		Host:          host,
		Group: federationRepositoryGroupResult{
			GroupID: selection.Group.GroupID,
			Name:    selection.Group.Name,
			Active:  selection.Group.Active,
			Agents:  append([]string(nil), selection.Group.Agents...),
		},
		Instance: selection.Instance,
	}, nil
}
