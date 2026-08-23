package federation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/identity"
)

const ProjectionSchemaVersion = 1

type Projection struct {
	SchemaVersion int                `json:"schema_version"`
	GeneratedAt   time.Time          `json:"generated_at"`
	Hosts         []HostProjection   `json:"hosts"`
	Repositories  []RepositoryGroup  `json:"repositories"`
	Issues        []CorrelationIssue `json:"correlation_issues,omitempty"`
}

type HostProjection struct {
	HostID     string    `json:"host_id"`
	Hostname   string    `json:"hostname"`
	ObservedAt time.Time `json:"observed_at"`
}

type RepositoryGroup struct {
	GroupID   string            `json:"group_id"`
	Name      string            `json:"name"`
	Active    bool              `json:"active"`
	Agents    []string          `json:"agents,omitempty"`
	Instances []SourcedInstance `json:"instances"`
}

type SourcedInstance struct {
	HostID     string    `json:"host_id"`
	Hostname   string    `json:"hostname"`
	ObservedAt time.Time `json:"observed_at"`
	RepositoryInstance
}

type CorrelationIssue struct {
	LeftInstanceID  string                      `json:"left_instance_id"`
	RightInstanceID string                      `json:"right_instance_id"`
	Outcome         identity.CorrelationOutcome `json:"outcome"`
	Reasons         []string                    `json:"reasons,omitempty"`
}

type Aggregator struct {
	now func() time.Time
}

func NewAggregator() *Aggregator {
	return &Aggregator{now: time.Now}
}

func (a *Aggregator) Aggregate(snapshots ...HostSnapshot) (Projection, error) {
	if a == nil {
		return Projection{}, errors.New("federation aggregator is required")
	}

	projection := Projection{
		SchemaVersion: ProjectionSchemaVersion,
		GeneratedAt:   a.now().UTC(),
	}

	current, err := currentSnapshots(snapshots)
	if err != nil {
		return Projection{}, err
	}

	var instances []SourcedInstance
	for _, snapshot := range current {
		projection.Hosts = append(projection.Hosts, HostProjection{
			HostID:     snapshot.HostID,
			Hostname:   snapshot.Hostname,
			ObservedAt: snapshot.ObservedAt,
		})
		for _, instance := range snapshot.Instances {
			instances = append(instances, SourcedInstance{
				HostID:             snapshot.HostID,
				Hostname:           snapshot.Hostname,
				ObservedAt:         snapshot.ObservedAt,
				RepositoryInstance: instance,
			})
		}
	}

	sort.Slice(instances, func(i, j int) bool {
		if instances[i].HostID != instances[j].HostID {
			return instances[i].HostID < instances[j].HostID
		}
		return instances[i].InstanceID < instances[j].InstanceID
	})

	groups := make([][]SourcedInstance, 0, len(instances))
	for _, candidate := range instances {
		eligible := make([]int, 0, 1)
		for groupIndex, group := range groups {
			allMatch := true
			for _, member := range group {
				result := identity.CorrelateRepositories(candidate.Fingerprint, member.Fingerprint)
				if result.Outcome != identity.CorrelationMatch {
					allMatch = false
					if result.Outcome == identity.CorrelationAmbiguous || result.Outcome == identity.CorrelationConflict {
						projection.Issues = appendIssue(projection.Issues, CorrelationIssue{
							LeftInstanceID:  member.InstanceID,
							RightInstanceID: candidate.InstanceID,
							Outcome:         result.Outcome,
							Reasons:         append([]string(nil), result.Reasons...),
						})
					}
				}
			}
			if allMatch {
				eligible = append(eligible, groupIndex)
			}
		}

		if len(eligible) == 1 {
			groups[eligible[0]] = append(groups[eligible[0]], candidate)
			continue
		}
		if len(eligible) > 1 {
			for _, groupIndex := range eligible {
				projection.Issues = appendIssue(projection.Issues, CorrelationIssue{
					LeftInstanceID:  groups[groupIndex][0].InstanceID,
					RightInstanceID: candidate.InstanceID,
					Outcome:         identity.CorrelationAmbiguous,
					Reasons:         []string{"candidate_matches_multiple_repository_groups"},
				})
			}
		}
		groups = append(groups, []SourcedInstance{candidate})
	}

	projection.Repositories = make([]RepositoryGroup, 0, len(groups))
	for _, group := range groups {
		projection.Repositories = append(projection.Repositories, buildRepositoryGroup(group))
	}
	sort.Slice(projection.Repositories, func(i, j int) bool {
		left, right := projection.Repositories[i], projection.Repositories[j]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.GroupID < right.GroupID
	})
	sort.Slice(projection.Issues, func(i, j int) bool {
		left, right := projection.Issues[i], projection.Issues[j]
		if left.LeftInstanceID != right.LeftInstanceID {
			return left.LeftInstanceID < right.LeftInstanceID
		}
		if left.RightInstanceID != right.RightInstanceID {
			return left.RightInstanceID < right.RightInstanceID
		}
		return left.Outcome < right.Outcome
	})
	return projection, nil
}

func currentSnapshots(snapshots []HostSnapshot) ([]HostSnapshot, error) {
	latest := make(map[string]HostSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		if err := snapshot.Validate(); err != nil {
			return nil, err
		}
		existing, ok := latest[snapshot.HostID]
		if !ok || snapshot.ObservedAt.After(existing.ObservedAt) {
			latest[snapshot.HostID] = snapshot
			continue
		}
		if snapshot.ObservedAt.Equal(existing.ObservedAt) && !reflect.DeepEqual(snapshot, existing) {
			return nil, fmt.Errorf("conflicting federation snapshots for Host %q at %s", snapshot.HostID, snapshot.ObservedAt.UTC().Format(time.RFC3339Nano))
		}
	}

	current := make([]HostSnapshot, 0, len(latest))
	for _, snapshot := range latest {
		current = append(current, snapshot)
	}
	sort.Slice(current, func(i, j int) bool {
		return current[i].HostID < current[j].HostID
	})
	return current, nil
}

func buildRepositoryGroup(instances []SourcedInstance) RepositoryGroup {
	members := append([]SourcedInstance(nil), instances...)
	sort.Slice(members, func(i, j int) bool {
		if members[i].HostID != members[j].HostID {
			return members[i].HostID < members[j].HostID
		}
		return members[i].InstanceID < members[j].InstanceID
	})

	group := RepositoryGroup{
		GroupID:   derivedGroupID(members),
		Instances: members,
	}
	agents := make(map[string]struct{})
	for _, instance := range members {
		if group.Name == "" && strings.TrimSpace(instance.Name) != "" {
			group.Name = instance.Name
		}
		if instance.Active {
			group.Active = true
		}
		for _, agent := range instance.Agents {
			if strings.TrimSpace(agent) != "" {
				agents[agent] = struct{}{}
			}
		}
	}
	group.Agents = make([]string, 0, len(agents))
	for agent := range agents {
		group.Agents = append(group.Agents, agent)
	}
	sort.Strings(group.Agents)
	return group
}

func derivedGroupID(instances []SourcedInstance) string {
	ids := make([]string, 0, len(instances))
	for _, instance := range instances {
		ids = append(ids, instance.InstanceID)
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, "\x00")))
	return "group:" + hex.EncodeToString(sum[:16])
}

func appendIssue(issues []CorrelationIssue, issue CorrelationIssue) []CorrelationIssue {
	left, right := issue.LeftInstanceID, issue.RightInstanceID
	if right < left {
		left, right = right, left
	}
	issue.LeftInstanceID = left
	issue.RightInstanceID = right
	for _, existing := range issues {
		if existing.LeftInstanceID == issue.LeftInstanceID && existing.RightInstanceID == issue.RightInstanceID && existing.Outcome == issue.Outcome {
			return issues
		}
	}
	return append(issues, issue)
}
