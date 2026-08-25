package controlplane

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/sergii/specview/internal/executionhistory"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/projectstate"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

type GetHostControlPlaneResult struct {
	SchemaVersion int                    `json:"schema_version"`
	Host          string                 `json:"host"`
	Intent        HostIntentSummary      `json:"intent"`
	Execution     HostExecutionSummary   `json:"execution"`
	Evidence      HostEvidenceSummary    `json:"evidence"`
	Acceptance    HostAcceptanceSummary  `json:"acceptance"`
	Attention     []HostAttentionSummary `json:"attention"`
}

type HostIntentSummary struct {
	ManagedRepositories int `json:"managed_repositories"`
	WorkItems           int `json:"work_items"`
	New                 int `json:"new"`
	InProgress          int `json:"in_progress"`
	Done                int `json:"done"`
	Invalid             int `json:"invalid"`
	Unavailable         int `json:"unavailable"`
}

type HostExecutionSummary struct {
	ActiveSessions     int                    `json:"active_sessions"`
	ActiveRepositories int                    `json:"active_repositories"`
	HasLatest          bool                   `json:"has_latest"`
	Latest             executionhistory.Entry `json:"latest"`
}

type HostEvidenceSummary struct {
	Total                int                       `json:"total"`
	Passed               int                       `json:"passed"`
	Failed               int                       `json:"failed"`
	Invalid              int                       `json:"invalid"`
	AffectedRepositories int                       `json:"affected_repositories"`
	Unavailable          int                       `json:"unavailable"`
	HasLatest            bool                      `json:"has_latest"`
	Latest               HostEvidenceLatestSummary `json:"latest"`
}

type HostEvidenceLatestSummary struct {
	RepositoryID   string                          `json:"repository_id"`
	RepositoryName string                          `json:"repository_name"`
	Entry          RepositoryEvidenceLatestSummary `json:"entry"`
}

type HostAcceptanceSummary struct {
	ConfiguredRepositories        int `json:"configured_repositories"`
	UnconfiguredRepositories      int `json:"unconfigured_repositories"`
	Accepted                      int `json:"accepted"`
	Waiting                       int `json:"waiting"`
	Blocked                       int `json:"blocked"`
	Unconfigured                  int `json:"unconfigured"`
	Invalid                       int `json:"invalid"`
	EvaluationPendingRepositories int `json:"evaluation_pending_repositories"`
	UnavailableRepositories       int `json:"unavailable_repositories"`
}

type HostAttentionSummary struct {
	RepositoryID   string    `json:"repository_id"`
	RepositoryName string    `json:"repository_name"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	Signals        []string  `json:"signals"`
}

func (r *Reader) GetHostControlPlane(ctx context.Context) (GetHostControlPlaneResult, error) {
	catalog, err := hoststate.OpenCatalog(r.statePath)
	if err != nil {
		return GetHostControlPlaneResult{}, err
	}
	return BuildHostControlPlane(ctx, catalog.Hostname(), catalog.Repositories(), r.sourceControl), nil
}

func BuildHostControlPlane(ctx context.Context, hostname string, repositories []hoststate.Repository, sourceControl sourcecontrol.Source) GetHostControlPlaneResult {
	if sourceControl == nil {
		sourceControl = sourcecontrol.DefaultService()
	}
	result := GetHostControlPlaneResult{
		SchemaVersion: SchemaVersion,
		Host:          hostname,
		Attention:     make([]HostAttentionSummary, 0),
	}

	history := executionhistory.Build(hostname, repositories)
	activeRepositories := make(map[string]struct{})
	for _, entry := range history.Entries {
		if !entry.Active {
			continue
		}
		result.Execution.ActiveSessions++
		activeRepositories[entry.RepositoryID] = struct{}{}
	}
	result.Execution.ActiveRepositories = len(activeRepositories)
	if len(history.Entries) > 0 {
		result.Execution.HasLatest = true
		result.Execution.Latest = history.Entries[0]
	}

	var latestEvidenceObservedAt time.Time
	for _, repository := range repositories {
		attention := HostAttentionSummary{
			RepositoryID:   repository.ID,
			RepositoryName: repository.Name,
			LastSeenAt:     repository.LastSeenAt,
		}

		project, err := projectstate.Resolve(repository.Root)
		if err != nil {
			result.Intent.Unavailable++
			result.Evidence.Unavailable++
			result.Acceptance.UnavailableRepositories++
			attention.Signals = append(attention.Signals, "Intent unavailable", "Evidence unavailable", "Acceptance unavailable")
			result.Attention = append(result.Attention, attention)
			continue
		}

		managed := project.Convention.Recognized && project.Convention.Supported
		if managed {
			result.Intent.ManagedRepositories++
		}

		items, itemErr := project.WorkItems()
		if itemErr != nil {
			if project.Convention.Recognized {
				result.Intent.Unavailable++
				attention.Signals = append(attention.Signals, "Intent unavailable")
			}
		} else {
			for _, item := range items {
				result.Intent.WorkItems++
				if item.Error != "" {
					result.Intent.Invalid++
					continue
				}
				switch item.Status {
				case specs.StatusNew:
					result.Intent.New++
				case specs.StatusInProgress:
					result.Intent.InProgress++
				case specs.StatusDone:
					result.Intent.Done++
				}
			}
			invalid := countHostInvalidArtifacts(items)
			if invalid > 0 {
				attention.Signals = append(attention.Signals, hostCountSignal(invalid, "invalid Intent item", "invalid Intent items"))
			}
		}

		evidenceOverview, evidenceErr := project.EvidenceOverview()
		if evidenceErr != nil {
			if managed {
				result.Evidence.Unavailable++
				attention.Signals = append(attention.Signals, "Evidence unavailable")
			}
		} else {
			result.Evidence.Total += evidenceOverview.Total
			result.Evidence.Passed += evidenceOverview.Passed
			result.Evidence.Failed += evidenceOverview.Failed
			result.Evidence.Invalid += evidenceOverview.Invalid
			if evidenceOverview.Failed > 0 || evidenceOverview.Invalid > 0 {
				result.Evidence.AffectedRepositories++
			}
			if evidenceOverview.Failed > 0 {
				attention.Signals = append(attention.Signals, hostCountSignal(evidenceOverview.Failed, "failed Evidence record", "failed Evidence records"))
			}
			if evidenceOverview.Invalid > 0 {
				attention.Signals = append(attention.Signals, hostCountSignal(evidenceOverview.Invalid, "invalid Evidence record", "invalid Evidence records"))
			}
			if len(evidenceOverview.Records) > 0 {
				record := evidenceOverview.Records[0]
				candidate := HostEvidenceLatestSummary{
					RepositoryID:   repository.ID,
					RepositoryName: repository.Name,
					Entry:          repositoryEvidenceLatestSummary(record),
				}
				if !result.Evidence.HasLatest || hostEvidenceLatestBefore(result.Evidence.Latest, latestEvidenceObservedAt, candidate, record.Record.ObservedAt) {
					result.Evidence.HasLatest = true
					result.Evidence.Latest = candidate
					latestEvidenceObservedAt = record.Record.ObservedAt
				}
			}
		}

		if project.Convention.Recognized {
			var gitContext sourcecontrol.GitContext
			if len(project.Policy.Required) > 0 {
				sourceContext, inspectErr := sourceControl.Inspect(ctx, repository.Root)
				if inspectErr != nil {
					result.Acceptance.UnavailableRepositories++
					attention.Signals = append(attention.Signals, "Acceptance unavailable")
					if len(attention.Signals) > 0 {
						result.Attention = append(result.Attention, attention)
					}
					continue
				}
				gitContext = sourceContext.Git
			}

			acceptanceOverview, acceptanceErr := project.AcceptanceOverview(gitContext)
			if acceptanceErr != nil {
				result.Acceptance.UnavailableRepositories++
				attention.Signals = append(attention.Signals, "Acceptance unavailable")
			} else {
				if acceptanceOverview.Configured {
					result.Acceptance.ConfiguredRepositories++
				} else {
					result.Acceptance.UnconfiguredRepositories++
				}
				result.Acceptance.Accepted += acceptanceOverview.Accepted
				result.Acceptance.Waiting += acceptanceOverview.Waiting
				result.Acceptance.Blocked += acceptanceOverview.Blocked
				result.Acceptance.Unconfigured += acceptanceOverview.Unconfigured
				result.Acceptance.Invalid += acceptanceOverview.Invalid
				if acceptanceOverview.EvaluationPending {
					result.Acceptance.EvaluationPendingRepositories++
				}
				if acceptanceOverview.Blocked > 0 {
					attention.Signals = append(attention.Signals, hostCountSignal(acceptanceOverview.Blocked, "blocked Acceptance item", "blocked Acceptance items"))
				}
				if acceptanceOverview.Waiting > 0 {
					attention.Signals = append(attention.Signals, hostCountSignal(acceptanceOverview.Waiting, "waiting Acceptance item", "waiting Acceptance items"))
				}
				if acceptanceOverview.Invalid > 0 {
					attention.Signals = append(attention.Signals, hostCountSignal(acceptanceOverview.Invalid, "invalid Acceptance item", "invalid Acceptance items"))
				}
			}
		}

		if len(attention.Signals) > 0 {
			result.Attention = append(result.Attention, attention)
		}
	}

	sort.SliceStable(result.Attention, func(i, j int) bool {
		left := result.Attention[i]
		right := result.Attention[j]
		if !left.LastSeenAt.Equal(right.LastSeenAt) {
			return left.LastSeenAt.After(right.LastSeenAt)
		}
		if left.RepositoryName != right.RepositoryName {
			return left.RepositoryName < right.RepositoryName
		}
		return left.RepositoryID < right.RepositoryID
	})
	return result
}

func countHostInvalidArtifacts(items []specs.Artifact) int {
	count := 0
	for _, item := range items {
		if item.Error != "" {
			count++
		}
	}
	return count
}

func hostCountSignal(count int, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return strconv.Itoa(count) + " " + label
}

func hostEvidenceLatestBefore(current HostEvidenceLatestSummary, currentObservedAt time.Time, candidate HostEvidenceLatestSummary, candidateObservedAt time.Time) bool {
	if !currentObservedAt.Equal(candidateObservedAt) {
		return currentObservedAt.Before(candidateObservedAt)
	}
	if current.Entry.Record.ID != candidate.Entry.Record.ID {
		return current.Entry.Record.ID > candidate.Entry.Record.ID
	}
	if current.RepositoryName != candidate.RepositoryName {
		return current.RepositoryName > candidate.RepositoryName
	}
	return current.RepositoryID > candidate.RepositoryID
}
