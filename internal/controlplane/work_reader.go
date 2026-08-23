package controlplane

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/evidence"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/projectstate"
	"github.com/sergii/specview/internal/revision"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/specs"
)

func (r *Reader) ListWorkItems(_ context.Context, repositoryID string) (ListWorkItemsResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return ListWorkItemsResult{}, err
	}
	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		return ListWorkItemsResult{}, err
	}
	items, err := project.WorkItems()
	if err != nil {
		return ListWorkItemsResult{}, err
	}

	result := ListWorkItemsResult{
		SchemaVersion:  SchemaVersion,
		Host:           catalog.Hostname(),
		RepositoryID:   repository.ID,
		RepositoryName: repository.Name,
		WorkItems:      make([]WorkItemListEntry, 0, len(items)),
	}
	for _, item := range items {
		result.WorkItems = append(result.WorkItems, workItemListEntry(item))
	}
	return result, nil
}

func (r *Reader) GetWorkItem(_ context.Context, repositoryID, workItemID string) (GetWorkItemResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return GetWorkItemResult{}, err
	}
	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		return GetWorkItemResult{}, err
	}
	item, err := project.FindWorkItem(workItemID)
	if err != nil {
		return GetWorkItemResult{}, err
	}
	return GetWorkItemResult{
		SchemaVersion:  SchemaVersion,
		Host:           catalog.Hostname(),
		RepositoryID:   repository.ID,
		RepositoryName: repository.Name,
		WorkItem:       workItemSummary(item),
	}, nil
}

func (r *Reader) GetEvidence(_ context.Context, repositoryID, workItemID string) (GetEvidenceResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return GetEvidenceResult{}, err
	}
	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		return GetEvidenceResult{}, err
	}
	item, err := project.FindWorkItem(workItemID)
	if err != nil {
		return GetEvidenceResult{}, err
	}
	records, err := project.Evidence(item.WorkItemID)
	if err != nil {
		return GetEvidenceResult{}, err
	}

	result := GetEvidenceResult{
		SchemaVersion:  SchemaVersion,
		Host:           catalog.Hostname(),
		RepositoryID:   repository.ID,
		RepositoryName: repository.Name,
		WorkItemID:     item.WorkItemID,
		Records:        make([]EvidenceSummary, 0, len(records)),
	}
	for _, record := range records {
		result.Records = append(result.Records, evidenceSummary(record))
	}
	return result, nil
}

func (r *Reader) GetAcceptance(ctx context.Context, repositoryID, workItemID string) (GetAcceptanceResult, error) {
	catalog, repository, err := r.repository(repositoryID)
	if err != nil {
		return GetAcceptanceResult{}, err
	}
	project, err := projectstate.Resolve(repository.Root)
	if err != nil {
		return GetAcceptanceResult{}, err
	}
	item, err := project.FindWorkItem(workItemID)
	if err != nil {
		return GetAcceptanceResult{}, err
	}

	var gitContext sourcecontrol.GitContext
	if len(project.Policy.Required) > 0 {
		sourceContext, inspectErr := r.sourceControl.Inspect(ctx, repository.Root)
		if inspectErr != nil {
			return GetAcceptanceResult{}, fmt.Errorf("source-control context unavailable: %w", inspectErr)
		}
		gitContext = sourceContext.Git
	}
	acceptanceResult, err := project.EvaluateAcceptance(item.WorkItemID, gitContext)
	if err != nil {
		return GetAcceptanceResult{}, err
	}

	return GetAcceptanceResult{
		SchemaVersion:     SchemaVersion,
		Host:              catalog.Hostname(),
		RepositoryID:      repository.ID,
		RepositoryName:    repository.Name,
		WorkItemID:        item.WorkItemID,
		Policy:            acceptancePolicySummary(project.Policy),
		Revision:          revisionSummary(acceptanceResult.Revision),
		Decision:          acceptanceDecisionSummary(acceptanceResult.Decision),
		EvidenceCount:     acceptanceResult.EvidenceCount,
		EvaluationPending: acceptanceResult.EvaluationPending,
	}, nil
}

func (r *Reader) repository(repositoryID string) (*hoststate.Catalog, hoststate.Repository, error) {
	catalog, err := hoststate.OpenCatalog(r.statePath)
	if err != nil {
		return nil, hoststate.Repository{}, err
	}

	id := strings.TrimSpace(repositoryID)
	if repository, ok := catalog.Find(id); ok {
		return catalog, repository, nil
	}

	// ListRepositories intentionally projects repositories that exist only in
	// current live Execution state, even before the Host runtime has persisted
	// them into the compatibility catalog. Every listed repository ID must remain
	// directly readable by the same control-plane Reader.
	sessions, sessionErr := r.executionSource.Sessions()
	if sessionErr == nil {
		for _, session := range sessions {
			root := normalizeRoot(session.RepositoryRoot)
			if root == "" || hoststate.RepositoryIDForRoot(root) != id {
				continue
			}
			return catalog, hoststate.Repository{
				ID:   id,
				Name: hoststate.RepositoryDisplayNameForRoot(root),
				Root: root,
			}, nil
		}
	}

	return nil, hoststate.Repository{}, fmt.Errorf("repository %q not found", repositoryID)
}

func workItemListEntry(item specs.Artifact) WorkItemListEntry {
	result := WorkItemListEntry{
		WorkItemID: item.WorkItemID,
		Kind:       string(item.Kind),
		Path:       item.Path,
		Title:      item.Title,
		Status:     string(item.Status),
		Error:      item.Error,
	}
	if !item.ModifiedAt.IsZero() {
		result.ModifiedAt = item.ModifiedAt.UTC().Format(time.RFC3339Nano)
	}
	return result
}

func workItemSummary(item specs.Artifact) WorkItemSummary {
	result := WorkItemSummary{
		ID:         item.ID,
		Kind:       string(item.Kind),
		Plane:      string(item.Plane),
		Role:       string(item.Role),
		WorkItemID: item.WorkItemID,
		Path:       item.Path,
		Title:      item.Title,
		Status:     string(item.Status),
		Body:       item.Body,
		Error:      item.Error,
		Relations:  make([]RelationSummary, 0, len(item.Relations)),
	}
	if !item.ModifiedAt.IsZero() {
		result.ModifiedAt = item.ModifiedAt.UTC().Format(time.RFC3339Nano)
	}
	for _, relation := range item.Relations {
		result.Relations = append(result.Relations, RelationSummary{Type: relation.Type, Target: relation.Target})
	}
	sort.Slice(result.Relations, func(i, j int) bool {
		if result.Relations[i].Type == result.Relations[j].Type {
			return result.Relations[i].Target < result.Relations[j].Target
		}
		return result.Relations[i].Type < result.Relations[j].Type
	})
	return result
}

func evidenceSummary(record evidence.Record) EvidenceSummary {
	result := EvidenceSummary{
		Version:    record.Version,
		ID:         record.ID,
		WorkItemID: record.WorkItemID,
		Revision:   record.Revision,
		Check:      record.Check,
		Kind:       string(record.Kind),
		Provider:   record.Provider,
		Result:     string(record.Result),
		ObservedAt: record.ObservedAt.UTC().Format(time.RFC3339Nano),
		Summary:    record.Summary,
		Path:       record.Path,
		Error:      record.Error,
	}
	if record.StartedAt != nil {
		result.StartedAt = record.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if record.FinishedAt != nil {
		result.FinishedAt = record.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	if len(record.Metrics) > 0 {
		result.Metrics = make(map[string]float64, len(record.Metrics))
		for key, value := range record.Metrics {
			result.Metrics[key] = value
		}
	}
	return result
}

func acceptancePolicySummary(policy acceptance.Policy) AcceptancePolicySummary {
	result := AcceptancePolicySummary{Required: make([]AcceptanceRequirementSummary, 0, len(policy.Required))}
	for _, requirement := range policy.Required {
		result.Required = append(result.Required, AcceptanceRequirementSummary{
			Check:        requirement.Check,
			AllowSkipped: requirement.AllowSkipped,
		})
	}
	return result
}

func revisionSummary(value revision.Resolution) RevisionSummary {
	return RevisionSummary{
		Revision:     value.Revision,
		WorktreePath: value.WorktreePath,
		Available:    value.Available,
		Reason:       value.Reason,
	}
}

func acceptanceDecisionSummary(decision acceptance.Decision) AcceptanceDecisionSummary {
	result := AcceptanceDecisionSummary{
		WorkItemID: decision.WorkItemID,
		Revision:   decision.Revision,
		State:      string(decision.State),
		Checks:     make([]AcceptanceCheckSummary, 0, len(decision.Checks)),
	}
	for _, check := range decision.Checks {
		result.Checks = append(result.Checks, AcceptanceCheckSummary{
			Check:      check.Check,
			State:      string(check.State),
			Provider:   check.Provider,
			EvidenceID: check.EvidenceID,
			Summary:    check.Summary,
		})
	}
	return result
}
