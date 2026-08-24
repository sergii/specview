package controlplane

import (
	"context"

	"github.com/sergii/specview/internal/executionhistory"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/projectstate"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

type GetRepositoryControlPlaneResult struct {
	SchemaVersion  int                                 `json:"schema_version"`
	Host           string                              `json:"host"`
	RepositoryID   string                              `json:"repository_id"`
	RepositoryName string                              `json:"repository_name"`
	Intent         RepositoryIntentSummary             `json:"intent"`
	Execution      RepositoryExecutionSummary          `json:"execution"`
	Evidence       RepositoryEvidenceOverviewSummary   `json:"evidence"`
	Acceptance     RepositoryAcceptanceOverviewSummary `json:"acceptance"`
}

type RepositoryIntentSummary struct {
	Total      int    `json:"total"`
	New        int    `json:"new"`
	InProgress int    `json:"in_progress"`
	Done       int    `json:"done"`
	Invalid    int    `json:"invalid"`
	Error      string `json:"error,omitempty"`
}

type RepositoryExecutionSummary struct {
	Active int                     `json:"active"`
	Latest *executionhistory.Entry `json:"latest,omitempty"`
	Error  string                  `json:"error,omitempty"`
}

type RepositoryEvidenceOverviewSummary struct {
	Total         int                              `json:"total"`
	Passed        int                              `json:"passed"`
	Failed        int                              `json:"failed"`
	Invalid       int                              `json:"invalid"`
	WorkItemError string                           `json:"work_item_error,omitempty"`
	Latest        *RepositoryEvidenceLatestSummary `json:"latest,omitempty"`
	Error         string                           `json:"error,omitempty"`
}

type RepositoryEvidenceLatestSummary struct {
	Record        EvidenceSummary `json:"record"`
	WorkItemPath  string          `json:"work_item_path,omitempty"`
	WorkItemTitle string          `json:"work_item_title,omitempty"`
}

type RepositoryAcceptanceOverviewSummary struct {
	Configured        bool            `json:"configured"`
	Revision          RevisionSummary `json:"revision"`
	EvidenceCount     int             `json:"evidence_count"`
	EvaluationPending bool            `json:"evaluation_pending"`
	Accepted          int             `json:"accepted"`
	Waiting           int             `json:"waiting"`
	Blocked           int             `json:"blocked"`
	Unconfigured      int             `json:"unconfigured"`
	Invalid           int             `json:"invalid"`
	Error             string          `json:"error,omitempty"`
}

func (r *Reader) GetRepositoryControlPlane(ctx context.Context, repositoryID string) (GetRepositoryControlPlaneResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return GetRepositoryControlPlaneResult{}, err
	}

	result := GetRepositoryControlPlaneResult{
		SchemaVersion:  SchemaVersion,
		Host:           catalog.Hostname(),
		RepositoryID:   repository.ID,
		RepositoryName: repository.Name,
	}

	history := executionhistory.Build(catalog.Hostname(), []hoststate.Repository{repository})
	for _, entry := range history.Entries {
		if entry.Active {
			result.Execution.Active++
		}
	}
	if len(history.Entries) > 0 {
		latest := history.Entries[0]
		result.Execution.Latest = &latest
	}

	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		message := err.Error()
		result.Intent.Error = message
		result.Evidence.Error = message
		result.Acceptance.Error = message
		return result, nil
	}

	items, err := project.WorkItems()
	if err != nil {
		result.Intent.Error = err.Error()
	} else {
		for _, item := range items {
			result.Intent.Total++
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
	}

	evidenceOverview, err := project.EvidenceOverview()
	if err != nil {
		result.Evidence.Error = err.Error()
	} else {
		result.Evidence.Total = evidenceOverview.Total
		result.Evidence.Passed = evidenceOverview.Passed
		result.Evidence.Failed = evidenceOverview.Failed
		result.Evidence.Invalid = evidenceOverview.Invalid
		result.Evidence.WorkItemError = evidenceOverview.WorkItemError
		if len(evidenceOverview.Records) > 0 {
			latest := repositoryEvidenceLatestSummary(evidenceOverview.Records[0])
			result.Evidence.Latest = &latest
		}
	}

	var gitContext sourcecontrol.GitContext
	if len(project.Policy.Required) > 0 {
		sourceContext, inspectErr := r.sourceControl.Inspect(ctx, repository.Root)
		if inspectErr != nil {
			result.Acceptance.Error = "source-control context unavailable: " + inspectErr.Error()
			return result, nil
		}
		gitContext = sourceContext.Git
	}
	acceptanceOverview, err := project.AcceptanceOverview(gitContext)
	if err != nil {
		result.Acceptance.Error = err.Error()
		return result, nil
	}
	result.Acceptance = repositoryAcceptanceOverviewSummary(acceptanceOverview)
	return result, nil
}

func repositoryEvidenceLatestSummary(value projectstate.EvidenceOverviewRecord) RepositoryEvidenceLatestSummary {
	return RepositoryEvidenceLatestSummary{
		Record:        evidenceSummary(value.Record),
		WorkItemPath:  value.WorkItemPath,
		WorkItemTitle: value.WorkItemTitle,
	}
}

func repositoryAcceptanceOverviewSummary(value projectstate.AcceptanceOverview) RepositoryAcceptanceOverviewSummary {
	return RepositoryAcceptanceOverviewSummary{
		Configured:        value.Configured,
		Revision:          revisionSummary(value.Revision),
		EvidenceCount:     value.EvidenceCount,
		EvaluationPending: value.EvaluationPending,
		Accepted:          value.Accepted,
		Waiting:           value.Waiting,
		Blocked:           value.Blocked,
		Unconfigured:      value.Unconfigured,
		Invalid:           value.Invalid,
	}
}
