package projectstate

import (
	"path/filepath"

	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/evidence"
	"github.com/sergii/specview/internal/revision"
	"github.com/sergii/specview/internal/sourcecontrol"
)

type AcceptanceOverview struct {
	Configured        bool
	Revision          revision.Resolution
	EvidenceCount     int
	EvaluationPending bool
	Accepted          int
	Waiting           int
	Blocked           int
	Unconfigured      int
	Invalid           int
	Items             []AcceptanceOverviewItem
}

type AcceptanceOverviewItem struct {
	WorkItemID        string
	Title             string
	Path              string
	State             acceptance.State
	EvidenceCount     int
	EvaluationPending bool
}

func (p Project) AcceptanceOverview(git sourcecontrol.GitContext) (AcceptanceOverview, error) {
	items, err := p.WorkItems()
	if err != nil {
		return AcceptanceOverview{}, err
	}

	overview := AcceptanceOverview{
		Configured: len(p.Policy.Required) > 0,
		Items:      make([]AcceptanceOverviewItem, 0, len(items)),
	}
	if !overview.Configured {
		for _, item := range items {
			if item.Error != "" {
				overview.Invalid++
				continue
			}
			overview.Unconfigured++
			overview.Items = append(overview.Items, AcceptanceOverviewItem{
				WorkItemID: item.WorkItemID,
				Title:      item.Title,
				Path:       item.Path,
				State:      acceptance.StateUnconfigured,
			})
		}
		return overview, nil
	}

	overview.Revision = revision.ResolveGit(p.ProjectRoot, git)
	store := evidence.NewStore(evidence.NewNativeAdapter(filepath.Join(p.ProjectRoot, ".specview", "evidence")))
	if err := store.Refresh(); err != nil {
		return AcceptanceOverview{}, err
	}
	recordsByWorkItem := make(map[string][]evidence.Record)
	for _, record := range store.All() {
		recordsByWorkItem[record.WorkItemID] = append(recordsByWorkItem[record.WorkItemID], record)
	}

	for _, item := range items {
		if item.Error != "" {
			overview.Invalid++
			continue
		}

		records := recordsByWorkItem[item.WorkItemID]
		entry := AcceptanceOverviewItem{
			WorkItemID:    item.WorkItemID,
			Title:         item.Title,
			Path:          item.Path,
			EvidenceCount: len(records),
		}
		overview.EvidenceCount += len(records)

		if !overview.Revision.Available {
			entry.State = acceptance.StateWaiting
			entry.EvaluationPending = true
			overview.Waiting++
			overview.EvaluationPending = true
			overview.Items = append(overview.Items, entry)
			continue
		}

		decision, err := acceptance.Evaluate(p.Policy, item.WorkItemID, overview.Revision.Revision, records)
		if err != nil {
			return AcceptanceOverview{}, err
		}
		entry.State = decision.State
		switch decision.State {
		case acceptance.StateAccepted:
			overview.Accepted++
		case acceptance.StateWaiting:
			overview.Waiting++
		case acceptance.StateBlocked:
			overview.Blocked++
		case acceptance.StateUnconfigured:
			overview.Unconfigured++
		}
		overview.Items = append(overview.Items, entry)
	}
	return overview, nil
}
