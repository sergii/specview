package web

import (
	"github.com/sergii/specview/internal/executionhistory"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/projectstate"
)

type repositoryControlPlaneSummary struct {
	Intent     repositoryControlPlaneIntent
	Execution  repositoryControlPlaneExecution
	Evidence   repositoryControlPlaneEvidence
	Acceptance repositoryControlPlaneAcceptance
}

type repositoryControlPlaneIntent struct {
	Total      int
	New        int
	InProgress int
	Done       int
	Invalid    int
	Error      string
}

type repositoryControlPlaneExecution struct {
	Active    int
	HasLatest bool
	Latest    executionhistory.Entry
	Error     string
}

type repositoryControlPlaneEvidence struct {
	Overview  projectstate.EvidenceOverview
	HasLatest bool
	Latest    projectstate.EvidenceOverviewRecord
	Error     string
}

type repositoryControlPlaneAcceptance struct {
	Overview projectstate.AcceptanceOverview
	Error    string
}

// ControlPlaneSummary composes existing read-only projections for the repository
// page. It owns no new state or authority; every facet keeps its native source.
func (data projectData) ControlPlaneSummary() repositoryControlPlaneSummary {
	summary := repositoryControlPlaneSummary{
		Intent: repositoryControlPlaneIntent{
			Total:      data.Total,
			New:        len(data.New),
			InProgress: len(data.InProgress),
			Done:       len(data.Done),
			Invalid:    len(data.Invalid),
		},
	}
	if data.DetectionError != "" {
		summary.Intent.Error = data.DetectionError
	} else if !data.Convention.Recognized {
		summary.Intent.Error = "No recognized specification pattern."
	} else if data.Unsupported {
		summary.Intent.Error = "The recognized specification adapter is not supported."
	}

	history := executionhistory.Build(data.Hostname, []hoststate.Repository{data.Repo})
	for _, entry := range history.Entries {
		if entry.Active {
			summary.Execution.Active++
		}
	}
	if len(history.Entries) > 0 {
		summary.Execution.HasLatest = true
		summary.Execution.Latest = history.Entries[0]
	}

	project, err := projectstate.Resolve(data.Repo.Root)
	if err != nil {
		summary.Evidence.Error = err.Error()
		summary.Acceptance.Error = err.Error()
		return summary
	}

	evidenceOverview, err := project.EvidenceOverview()
	if err != nil {
		summary.Evidence.Error = err.Error()
	} else {
		summary.Evidence.Overview = evidenceOverview
		if len(evidenceOverview.Records) > 0 {
			summary.Evidence.HasLatest = true
			summary.Evidence.Latest = evidenceOverview.Records[0]
		}
	}

	acceptanceOverview, err := project.AcceptanceOverview(data.SourceControl.Git)
	if err != nil {
		summary.Acceptance.Error = err.Error()
	} else {
		summary.Acceptance.Overview = acceptanceOverview
	}
	return summary
}
