package evidence

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordRequiresRevision(t *testing.T) {
	finished := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	record := Record{
		Version:    1,
		ID:         "ATS-003-rspec",
		WorkItemID: "ATS-003",
		Check:      "unit-tests",
		Kind:       KindTest,
		Provider:   "rspec",
		Result:     ResultPassed,
		FinishedAt: &finished,
		ObservedAt: finished,
	}
	if err := record.Validate(); err == nil {
		t.Fatal("expected missing revision validation error")
	}
}

func TestAcceptanceEligibleRequiresMatchingRevisionAndPass(t *testing.T) {
	finished := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	record := Record{
		Version:    1,
		ID:         "ATS-003-rspec",
		WorkItemID: "ATS-003",
		Revision:   "git:abc123",
		Check:      "unit-tests",
		Kind:       KindTest,
		Provider:   "rspec",
		Result:     ResultPassed,
		FinishedAt: &finished,
		ObservedAt: finished,
	}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	if !record.AcceptanceEligible("git:abc123") {
		t.Fatal("matching passed evidence should be acceptance eligible")
	}
	if record.AcceptanceEligible("git:def456") {
		t.Fatal("stale evidence must not be acceptance eligible")
	}
	record.Result = ResultSkipped
	if record.AcceptanceEligible("git:abc123") {
		t.Fatal("skipped evidence must not be equivalent to passed")
	}
}

func TestTerminalEvidenceRequiresFinishedAt(t *testing.T) {
	record := Record{
		Version:    1,
		ID:         "ATS-003-brakeman",
		WorkItemID: "ATS-003",
		Revision:   "git:abc123",
		Check:      "security",
		Kind:       KindSecurity,
		Provider:   "brakeman",
		Result:     ResultFailed,
		ObservedAt: time.Now().UTC(),
	}
	if err := record.Validate(); err == nil {
		t.Fatal("expected terminal result to require finished_at")
	}
}

func TestNativeAdapterScansValidAndInvalidEvidence(t *testing.T) {
	root := t.TempDir()
	valid := `{
  "version": 1,
  "id": "ATS-003-rspec",
  "work_item_id": "ATS-003",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "rspec",
  "result": "passed",
  "started_at": "2026-08-21T11:59:52Z",
  "finished_at": "2026-08-21T12:00:00Z",
  "observed_at": "2026-08-21T12:00:00Z",
  "summary": "184 examples, 0 failures",
  "metrics": {"examples": 184, "failures": 0}
}`
	if err := os.WriteFile(filepath.Join(root, "rspec.json"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	invalid := `{
  "version": 1,
  "id": "ATS-003-rubocop",
  "work_item_id": "ATS-003",
  "check": "lint",
  "kind": "lint",
  "provider": "rubocop",
  "result": "passed",
  "finished_at": "2026-08-21T12:00:00Z",
  "observed_at": "2026-08-21T12:00:00Z"
}`
	if err := os.WriteFile(filepath.Join(root, "rubocop.json"), []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "writer.tmp"), []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("ignore me"), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewNativeAdapter(root)
	records, err := adapter.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 JSON evidence records, got %#v", records)
	}
	if records[0].ID != "ATS-003-rspec" || records[0].Error != "" || records[0].Result != ResultPassed {
		t.Fatalf("unexpected valid record: %#v", records[0])
	}
	if records[1].ID != "ATS-003-rubocop" || records[1].Error == "" {
		t.Fatalf("invalid record should remain observable: %#v", records[1])
	}
	if roots := adapter.WatchRoots(); len(roots) != 1 || roots[0] != filepath.Clean(root) {
		t.Fatalf("unexpected watch roots: %#v", roots)
	}
}

func TestNativeAdapterRejectsUnknownFields(t *testing.T) {
	root := t.TempDir()
	data := `{
  "version": 1,
  "id": "ATS-003-rspec",
  "work_item_id": "ATS-003",
  "revision": "git:abc123",
  "check": "unit-tests",
  "kind": "test",
  "provider": "rspec",
  "result": "passed",
  "finished_at": "2026-08-21T12:00:00Z",
  "observed_at": "2026-08-21T12:00:00Z",
  "requried": true
}`
	if err := os.WriteFile(filepath.Join(root, "typo.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	records, err := NewNativeAdapter(root).Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Error == "" {
		t.Fatalf("unknown field must surface as validation error: %#v", records)
	}
}

type stubAdapter struct {
	records []Record
}

func (a stubAdapter) Name() string            { return "stub" }
func (a stubAdapter) Scan() ([]Record, error) { return a.records, nil }
func (a stubAdapter) WatchRoots() []string    { return nil }

func TestStoreUsesEvidenceAdapterBoundary(t *testing.T) {
	store := NewStore(stubAdapter{records: []Record{
		{ID: "a", WorkItemID: "ATS-001"},
		{ID: "b", WorkItemID: "ATS-002"},
		{ID: "c", WorkItemID: "ATS-001"},
	}})
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	if store.AdapterName() != "stub" {
		t.Fatalf("unexpected adapter name %q", store.AdapterName())
	}
	if got := store.ForWorkItem("ATS-001"); len(got) != 2 {
		t.Fatalf("expected 2 ATS-001 records, got %#v", got)
	}
}
