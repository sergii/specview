package projectstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceOverviewKeepsValidAndInvalidNativeRecordsVisible(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root)
	evidenceRoot := filepath.Join(root, ".specview", "evidence")
	if err := os.MkdirAll(evidenceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := `{
  "version": 1,
  "id": "H18-unit-tests",
  "work_item_id": "H18",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "go-test",
  "result": "passed",
  "finished_at": "2026-08-24T12:00:00Z",
  "observed_at": "2026-08-24T12:00:00Z",
  "summary": "tests passed"
}
`
	invalid := `{
  "version": 1,
  "id": "H18-lint-invalid",
  "work_item_id": "H18",
  "revision": "git:abc123",
  "check": "lint",
  "kind": "lint",
  "provider": "fixture",
  "result": "failed",
  "finished_at": "2026-08-24T11:00:00Z"
}
`
	if err := os.WriteFile(filepath.Join(evidenceRoot, "valid.json"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "invalid.json"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}

	project, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	overview, err := project.EvidenceOverview()
	if err != nil {
		t.Fatal(err)
	}
	if overview.Total != 2 || overview.Passed != 1 || overview.Failed != 0 || overview.Invalid != 1 {
		t.Fatalf("unexpected evidence counts: %#v", overview)
	}
	if len(overview.Records) != 2 {
		t.Fatalf("records = %d, want 2", len(overview.Records))
	}
	if overview.Records[0].Record.ID != "H18-unit-tests" || overview.Records[0].WorkItemPath != "H18.md" || overview.Records[0].WorkItemTitle != "H18 MCP" {
		t.Fatalf("valid record not enriched from Intent: %#v", overview.Records[0])
	}
	if overview.Records[1].Record.Error == "" {
		t.Fatalf("invalid native evidence was not preserved: %#v", overview.Records[1])
	}
}

func TestEvidenceOverviewSurvivesUnavailableIntentProjection(t *testing.T) {
	root := t.TempDir()
	evidenceRoot := filepath.Join(root, ".specview", "evidence")
	if err := os.MkdirAll(evidenceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	record := `{
  "version": 1,
  "id": "orphan-check",
  "work_item_id": "UNKNOWN",
  "revision": "git:def456",
  "check": "contract",
  "kind": "contract",
  "provider": "fixture",
  "result": "passed",
  "finished_at": "2026-08-24T13:00:00Z",
  "observed_at": "2026-08-24T13:00:00Z"
}
`
	if err := os.WriteFile(filepath.Join(evidenceRoot, "orphan.json"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}

	project, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	overview, err := project.EvidenceOverview()
	if err != nil {
		t.Fatal(err)
	}
	if overview.Total != 1 || overview.WorkItemError == "" || overview.Records[0].Record.ID != "orphan-check" || overview.Records[0].WorkItemPath != "" {
		t.Fatalf("evidence must remain visible without Intent projection: %#v", overview)
	}
}
