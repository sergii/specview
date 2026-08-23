package web

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/evidence"
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
	data := workItemAcceptanceData{}
	projectRoot := filepath.Clean(repoRoot)
	policy := acceptance.Policy{}

	if _, err := os.Stat(filepath.Join(repoRoot, config.FileName)); err == nil {
		cfg, loadErr := config.Load(repoRoot)
		if loadErr != nil {
			data.Error = loadErr.Error()
			return data
		}
		projectRoot = cfg.ResolveProjectRoot(repoRoot)
		policy = cfg.AcceptancePolicy()
	} else if !errors.Is(err, os.ErrNotExist) {
		data.Error = err.Error()
		return data
	}

	if len(policy.Required) == 0 {
		decision, err := acceptance.Evaluate(policy, "", "", nil)
		if err != nil {
			data.Error = err.Error()
			return data
		}
		data.Decision = decision
		return data
	}

	data.Revision = revision.ResolveGit(projectRoot, git)

	store := evidence.NewStore(evidence.NewNativeAdapter(filepath.Join(projectRoot, ".specview", "evidence")))
	if err := store.Refresh(); err != nil {
		data.Error = err.Error()
		return data
	}
	records := store.ForWorkItem(item.WorkItemID)
	data.EvidenceCount = len(records)

	if !data.Revision.Available {
		data.Decision = acceptance.Decision{
			WorkItemID: item.WorkItemID,
			State:      acceptance.StateWaiting,
		}
		data.EvaluationPending = true
		return data
	}

	decision, err := acceptance.Evaluate(policy, item.WorkItemID, data.Revision.Revision, records)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	data.Decision = decision
	return data
}
