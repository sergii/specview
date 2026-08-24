package projectstate

import (
	"path/filepath"
	"sort"

	"github.com/sergii/specview/internal/evidence"
)

type EvidenceOverview struct {
	Total         int
	Passed        int
	Failed        int
	Invalid       int
	WorkItemError string
	Records       []EvidenceOverviewRecord
}

type EvidenceOverviewRecord struct {
	Record        evidence.Record
	WorkItemPath  string
	WorkItemTitle string
}

func (p Project) EvidenceOverview() (EvidenceOverview, error) {
	store := evidence.NewStore(evidence.NewNativeAdapter(filepath.Join(p.ProjectRoot, ".specview", "evidence")))
	if err := store.Refresh(); err != nil {
		return EvidenceOverview{}, err
	}

	overview := EvidenceOverview{}
	workItemsByID := make(map[string]struct {
		Path  string
		Title string
	})
	if items, err := p.WorkItems(); err != nil {
		overview.WorkItemError = err.Error()
	} else {
		for _, item := range items {
			if _, exists := workItemsByID[item.WorkItemID]; exists {
				continue
			}
			workItemsByID[item.WorkItemID] = struct {
				Path  string
				Title string
			}{Path: item.Path, Title: item.Title}
		}
	}

	for _, record := range store.All() {
		entry := EvidenceOverviewRecord{Record: record}
		if item, ok := workItemsByID[record.WorkItemID]; ok {
			entry.WorkItemPath = item.Path
			entry.WorkItemTitle = item.Title
		}
		overview.Records = append(overview.Records, entry)
		overview.Total++
		if record.Error != "" {
			overview.Invalid++
			continue
		}
		switch record.Result {
		case evidence.ResultPassed:
			overview.Passed++
		case evidence.ResultFailed, evidence.ResultError:
			overview.Failed++
		}
	}

	sort.SliceStable(overview.Records, func(i, j int) bool {
		left := overview.Records[i].Record
		right := overview.Records[j].Record
		if !left.ObservedAt.Equal(right.ObservedAt) {
			return left.ObservedAt.After(right.ObservedAt)
		}
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		return left.Path < right.Path
	})
	return overview, nil
}
