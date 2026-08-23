package controlplane

type GetWorkItemResult struct {
	SchemaVersion  int             `json:"schema_version"`
	Host           string          `json:"host"`
	RepositoryID   string          `json:"repository_id"`
	RepositoryName string          `json:"repository_name"`
	WorkItem       WorkItemSummary `json:"work_item"`
	Warnings       []string        `json:"warnings,omitempty"`
}

type WorkItemSummary struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Plane      string            `json:"plane"`
	Role       string            `json:"role"`
	WorkItemID string            `json:"work_item_id"`
	Path       string            `json:"path"`
	Title      string            `json:"title"`
	Status     string            `json:"status"`
	ModifiedAt string            `json:"modified_at,omitempty"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
	Relations  []RelationSummary `json:"relations,omitempty"`
}

type RelationSummary struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

type GetEvidenceResult struct {
	SchemaVersion  int               `json:"schema_version"`
	Host           string            `json:"host"`
	RepositoryID   string            `json:"repository_id"`
	RepositoryName string            `json:"repository_name"`
	WorkItemID     string            `json:"work_item_id"`
	Records        []EvidenceSummary `json:"records"`
	Warnings       []string          `json:"warnings,omitempty"`
}

type EvidenceSummary struct {
	Version    int                `json:"version"`
	ID         string             `json:"id"`
	WorkItemID string             `json:"work_item_id"`
	Revision   string             `json:"revision"`
	Check      string             `json:"check"`
	Kind       string             `json:"kind"`
	Provider   string             `json:"provider"`
	Result     string             `json:"result"`
	StartedAt  string             `json:"started_at,omitempty"`
	FinishedAt string             `json:"finished_at,omitempty"`
	ObservedAt string             `json:"observed_at"`
	Summary    string             `json:"summary,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
	Path       string             `json:"path,omitempty"`
	Error      string             `json:"error,omitempty"`
}

type GetAcceptanceResult struct {
	SchemaVersion     int                       `json:"schema_version"`
	Host              string                    `json:"host"`
	RepositoryID      string                    `json:"repository_id"`
	RepositoryName    string                    `json:"repository_name"`
	WorkItemID        string                    `json:"work_item_id"`
	Policy            AcceptancePolicySummary   `json:"policy"`
	Revision          RevisionSummary           `json:"revision"`
	Decision          AcceptanceDecisionSummary `json:"decision"`
	EvidenceCount     int                       `json:"evidence_count"`
	EvaluationPending bool                      `json:"evaluation_pending"`
	Warnings          []string                  `json:"warnings,omitempty"`
}

type AcceptancePolicySummary struct {
	Required []AcceptanceRequirementSummary `json:"required"`
}

type AcceptanceRequirementSummary struct {
	Check        string `json:"check"`
	AllowSkipped bool   `json:"allow_skipped"`
}

type RevisionSummary struct {
	Revision     string `json:"revision,omitempty"`
	WorktreePath string `json:"worktree_path,omitempty"`
	Available    bool   `json:"available"`
	Reason       string `json:"reason,omitempty"`
}

type AcceptanceDecisionSummary struct {
	WorkItemID string                   `json:"work_item_id"`
	Revision   string                   `json:"revision,omitempty"`
	State      string                   `json:"state"`
	Checks     []AcceptanceCheckSummary `json:"checks"`
}

type AcceptanceCheckSummary struct {
	Check      string `json:"check"`
	State      string `json:"state"`
	Provider   string `json:"provider,omitempty"`
	EvidenceID string `json:"evidence_id,omitempty"`
	Summary    string `json:"summary,omitempty"`
}
