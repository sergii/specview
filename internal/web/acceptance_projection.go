package web

import (
	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/projectstate"
	"github.com/sergii/specview/internal/revision"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

type workItemAcceptanceData struct {
	Decision          acceptance.Decision
	Revision          revision.Resolution
	EvidenceCount     int
	EvaluationPending bool
	Error             string
}

func (d workItemAcceptanceData) RevisionLabel() string {
	if d.Revision.Available {
		return d.Revision.Revision
	}
	switch d.Revision.Reason {
	case revision.ReasonDirtyWorktree:
		return "unavailable: dirty worktree"
	case revision.ReasonHeadUnavailable:
		return "unavailable: Git HEAD unavailable"
	case revision.ReasonWorktreeNotFound:
		return "unavailable: worktree not found"
	default:
		return "unavailable"
	}
}

func loadWorkItemAcceptance(repoRoot string, item specs.Artifact, git sourcecontrol.GitContext) workItemAcceptanceData {
	project, err := projectstate.Resolve(repoRoot)
	if err != nil {
		return workItemAcceptanceData{Error: err.Error()}
	}
	result, err := project.EvaluateAcceptance(item.WorkItemID, git)
	if err != nil {
		return workItemAcceptanceData{Error: err.Error()}
	}
	return workItemAcceptanceData{
		Decision:          result.Decision,
		Revision:          result.Revision,
		EvidenceCount:     result.EvidenceCount,
		EvaluationPending: result.EvaluationPending,
	}
}
