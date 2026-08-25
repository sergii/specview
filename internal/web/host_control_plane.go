package web

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

type hostControlPlaneSummary struct {
	Intent     hostControlPlaneIntent
	Execution  hostControlPlaneExecution
	Evidence   hostControlPlaneEvidence
	Acceptance hostControlPlaneAcceptance
	Attention  []hostControlPlaneAttention
}

type hostControlPlaneIntent struct {
	ManagedRepositories int
	WorkItems           int
	New                 int
	InProgress          int
	Done                int
	Invalid             int
	Unavailable         int
}

type hostControlPlaneExecution struct {
	ActiveSessions     int
	ActiveRepositories int
	HasLatest          bool
	Latest             executionhistory.Entry
}

type hostControlPlaneEvidence struct {
	Total                int
	Passed               int
	Failed               int
	Invalid              int
	AffectedRepositories int
	Unavailable          int
	HasLatest            bool
	Latest               hostControlPlaneLatestEvidence
}

type hostControlPlaneLatestEvidence struct {
	RepositoryID   string
	RepositoryName string
	Entry          projectstate.EvidenceOverviewRecord
}

type hostControlPlaneAcceptance struct {
	ConfiguredRepositories        int
	UnconfiguredRepositories      int
	Accepted                      int
	Waiting                       int
	Blocked                       int
	Unconfigured                  int
	Invalid                       int
	EvaluationPendingRepositories int
	UnavailableRepositories       int
}

type hostControlPlaneAttention struct {
	RepositoryID   string
	RepositoryName string
	LastSeenAt     time.Time
	Signals        []string
}

func (s *HostServer) hostControlPlane(ctx context.Context, repositories []hoststate.Repository) hostControlPlaneSummary {
	summary := hostControlPlaneSummary{}

	history := executionhistory.Build(s.catalog.Hostname(), repositories)
	activeRepositories := make(map[string]struct{})
	for _, entry := range history.Entries {
		if !entry.Active {
			continue
		}
		summary.Execution.ActiveSessions++
		activeRepositories[entry.RepositoryID] = struct{}{}
	}
	summary.Execution.ActiveRepositories = len(activeRepositories)
	if len(history.Entries) > 0 {
		summary.Execution.HasLatest = true
		summary.Execution.Latest = history.Entries[0]
	}

	for _, repository := range repositories {
		attention := hostControlPlaneAttention{
			RepositoryID:   repository.ID,
			RepositoryName: repository.Name,
			LastSeenAt:     repository.LastSeenAt,
		}

		project, err := projectstate.Resolve(repository.Root)
		if err != nil {
			summary.Intent.Unavailable++
			summary.Evidence.Unavailable++
			summary.Acceptance.UnavailableRepositories++
			attention.Signals = append(attention.Signals, "Intent unavailable", "Evidence unavailable", "Acceptance unavailable")
			summary.Attention = append(summary.Attention, attention)
			continue
		}

		managed := project.Convention.Recognized && project.Convention.Supported
		if managed {
			summary.Intent.ManagedRepositories++
		}

		items, itemErr := project.WorkItems()
		if itemErr != nil {
			if project.Convention.Recognized {
				summary.Intent.Unavailable++
				attention.Signals = append(attention.Signals, "Intent unavailable")
			}
		} else {
			for _, item := range items {
				summary.Intent.WorkItems++
				if item.Error != "" {
					summary.Intent.Invalid++
					continue
				}
				switch item.Status {
				case specs.StatusNew:
					summary.Intent.New++
				case specs.StatusInProgress:
					summary.Intent.InProgress++
				case specs.StatusDone:
					summary.Intent.Done++
				}
			}
			invalid := countInvalidArtifacts(items)
			if invalid > 0 {
				attention.Signals = append(attention.Signals, countSignal(invalid, "invalid Intent item", "invalid Intent items"))
			}
		}

		evidenceOverview, evidenceErr := project.EvidenceOverview()
		if evidenceErr != nil {
			if managed {
				summary.Evidence.Unavailable++
				attention.Signals = append(attention.Signals, "Evidence unavailable")
			}
		} else {
			summary.Evidence.Total += evidenceOverview.Total
			summary.Evidence.Passed += evidenceOverview.Passed
			summary.Evidence.Failed += evidenceOverview.Failed
			summary.Evidence.Invalid += evidenceOverview.Invalid
			if evidenceOverview.Failed > 0 || evidenceOverview.Invalid > 0 {
				summary.Evidence.AffectedRepositories++
			}
			if evidenceOverview.Failed > 0 {
				attention.Signals = append(attention.Signals, countSignal(evidenceOverview.Failed, "failed Evidence record", "failed Evidence records"))
			}
			if evidenceOverview.Invalid > 0 {
				attention.Signals = append(attention.Signals, countSignal(evidenceOverview.Invalid, "invalid Evidence record", "invalid Evidence records"))
			}
			if len(evidenceOverview.Records) > 0 {
				candidate := hostControlPlaneLatestEvidence{
					RepositoryID:   repository.ID,
					RepositoryName: repository.Name,
					Entry:          evidenceOverview.Records[0],
				}
				if !summary.Evidence.HasLatest || evidenceLatestBefore(summary.Evidence.Latest, candidate) {
					summary.Evidence.HasLatest = true
					summary.Evidence.Latest = candidate
				}
			}
		}

		if project.Convention.Recognized {
			var gitContext sourcecontrol.GitContext
			if len(project.Policy.Required) > 0 {
				sourceContext, inspectErr := s.sourceControl.Inspect(ctx, repository.Root)
				if inspectErr != nil {
					summary.Acceptance.UnavailableRepositories++
					attention.Signals = append(attention.Signals, "Acceptance unavailable")
					if len(attention.Signals) > 0 {
						summary.Attention = append(summary.Attention, attention)
					}
					continue
				}
				gitContext = sourceContext.Git
			}

			acceptanceOverview, acceptanceErr := project.AcceptanceOverview(gitContext)
			if acceptanceErr != nil {
				summary.Acceptance.UnavailableRepositories++
				attention.Signals = append(attention.Signals, "Acceptance unavailable")
			} else {
				if acceptanceOverview.Configured {
					summary.Acceptance.ConfiguredRepositories++
				} else {
					summary.Acceptance.UnconfiguredRepositories++
				}
				summary.Acceptance.Accepted += acceptanceOverview.Accepted
				summary.Acceptance.Waiting += acceptanceOverview.Waiting
				summary.Acceptance.Blocked += acceptanceOverview.Blocked
				summary.Acceptance.Unconfigured += acceptanceOverview.Unconfigured
				summary.Acceptance.Invalid += acceptanceOverview.Invalid
				if acceptanceOverview.EvaluationPending {
					summary.Acceptance.EvaluationPendingRepositories++
				}
				if acceptanceOverview.Blocked > 0 {
					attention.Signals = append(attention.Signals, countSignal(acceptanceOverview.Blocked, "blocked Acceptance item", "blocked Acceptance items"))
				}
				if acceptanceOverview.Waiting > 0 {
					attention.Signals = append(attention.Signals, countSignal(acceptanceOverview.Waiting, "waiting Acceptance item", "waiting Acceptance items"))
				}
				if acceptanceOverview.Invalid > 0 {
					attention.Signals = append(attention.Signals, countSignal(acceptanceOverview.Invalid, "invalid Acceptance item", "invalid Acceptance items"))
				}
			}
		}

		if len(attention.Signals) > 0 {
			summary.Attention = append(summary.Attention, attention)
		}
	}

	sort.SliceStable(summary.Attention, func(i, j int) bool {
		left := summary.Attention[i]
		right := summary.Attention[j]
		if !left.LastSeenAt.Equal(right.LastSeenAt) {
			return left.LastSeenAt.After(right.LastSeenAt)
		}
		if left.RepositoryName != right.RepositoryName {
			return left.RepositoryName < right.RepositoryName
		}
		return left.RepositoryID < right.RepositoryID
	})
	return summary
}

func countInvalidArtifacts(items []specs.Artifact) int {
	count := 0
	for _, item := range items {
		if item.Error != "" {
			count++
		}
	}
	return count
}

func countSignal(count int, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return strconv.Itoa(count) + " " + label
}

func evidenceLatestBefore(current, candidate hostControlPlaneLatestEvidence) bool {
	left := current.Entry.Record
	right := candidate.Entry.Record
	if !left.ObservedAt.Equal(right.ObservedAt) {
		return left.ObservedAt.Before(right.ObservedAt)
	}
	if left.ID != right.ID {
		return left.ID > right.ID
	}
	if current.RepositoryName != candidate.RepositoryName {
		return current.RepositoryName > candidate.RepositoryName
	}
	return current.RepositoryID > candidate.RepositoryID
}
