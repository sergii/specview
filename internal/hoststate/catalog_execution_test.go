package hoststate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/config"
)

func logicalTestDetector(string) (config.Convention, error) {
	return config.Convention{}, nil
}

func TestCatalogV1MigratesToLegacyProcessDiagnosticsAndFlushesV2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	v1 := `{
  "version": 1,
  "repositories": [{
    "id": "repo-v1",
    "name": "specview",
    "root": "/work/specview",
    "first_seen_at": "2026-08-24T00:00:00Z",
    "last_seen_at": "2026-08-24T00:10:00Z",
    "convention": {},
    "sessions": [{
      "id": "session-v1",
      "agent": "Codex",
      "pid": 4242,
      "started_at": "2026-08-24T00:01:00Z",
      "last_seen_at": "2026-08-24T00:10:00Z",
      "active": true
    }]
  }]
}`
	if err := os.WriteFile(path, []byte(v1), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog, err := openCatalog(path, logicalTestDetector)
	if err != nil {
		t.Fatal(err)
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 || len(repositories[0].Sessions) != 1 {
		t.Fatalf("unexpected migrated catalog: %#v", repositories)
	}
	session := repositories[0].Sessions[0]
	if session.ID != "session-v1" || session.IdentityKind != SessionIdentityLegacyPID || session.Adapter != "codex" {
		t.Fatalf("unexpected migrated session identity: %#v", session)
	}
	if len(session.ProcessIDs) != 1 || session.ProcessIDs[0] != 4242 {
		t.Fatalf("unexpected migrated process diagnostics: %#v", session.ProcessIDs)
	}
	if !catalog.materialDirty {
		t.Fatal("v1 load must remain material-dirty until canonical v2 persistence")
	}

	if err := catalog.Flush(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"pid"`) {
		t.Fatalf("catalog v2 must not persist singular pid field: %s", data)
	}
	var header struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		t.Fatal(err)
	}
	if header.Version != catalogVersion {
		t.Fatalf("catalog version = %d, want %d", header.Version, catalogVersion)
	}
}

func TestCatalogLogicalSessionPersistsOnceAcrossMultipleProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	catalog, err := openCatalog(path, logicalTestDetector)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	execution := ExecutionSession{
		Adapter:        "codex",
		ID:             "execution-specview",
		Agent:          "Codex",
		CWD:            "/work/specview",
		RepositoryRoot: "/work/specview",
		WorktreeRoot:   "/work/specview",
		ProcessIDs:     []int{30, 10, 20, 20},
	}
	if changed, err := catalog.ObserveExecutions([]ExecutionSession{execution}, now); err != nil || !changed {
		t.Fatalf("ObserveExecutions changed=%v err=%v", changed, err)
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 || len(repositories[0].Sessions) != 1 {
		t.Fatalf("logical execution was not persisted as one session: %#v", repositories)
	}
	session := repositories[0].Sessions[0]
	if session.ID != execution.ID || session.IdentityKind != SessionIdentityLogical {
		t.Fatalf("unexpected logical session: %#v", session)
	}
	wantPIDs := []int{10, 20, 30}
	if !equalIntSlices(session.ProcessIDs, wantPIDs) {
		t.Fatalf("process IDs = %#v, want %#v", session.ProcessIDs, wantPIDs)
	}
}

func TestCatalogLogicalProcessChurnPreservesSessionIdentity(t *testing.T) {
	catalog, err := openCatalog(filepath.Join(t.TempDir(), "catalog.json"), logicalTestDetector)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	first := ExecutionSession{
		Adapter:        "codex",
		ID:             "execution-specview",
		Agent:          "Codex",
		CWD:            "/work/specview",
		RepositoryRoot: "/work/specview",
		WorktreeRoot:   "/work/specview",
		ProcessIDs:     []int{10, 20},
	}
	if _, err := catalog.ObserveExecutions([]ExecutionSession{first}, started); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ProcessIDs = []int{20, 30}
	if _, err := catalog.ObserveExecutions([]ExecutionSession{second}, started.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	sessions := catalog.Repositories()[0].Sessions
	if len(sessions) != 1 || sessions[0].ID != first.ID || !sessions[0].StartedAt.Equal(started) {
		t.Fatalf("process churn created identity churn: %#v", sessions)
	}
	if !equalIntSlices(sessions[0].ProcessIDs, []int{20, 30}) {
		t.Fatalf("process diagnostics were not refreshed: %#v", sessions[0].ProcessIDs)
	}
}

func TestCatalogCollapsesActiveV1FragmentsIntoLogicalSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	v1 := `{
  "version": 1,
  "repositories": [{
    "id": "repo-legacy",
    "name": "specview",
    "root": "/work/specview",
    "first_seen_at": "2026-08-24T00:00:00Z",
    "last_seen_at": "2026-08-24T00:10:00Z",
    "convention": {},
    "sessions": [
      {"id":"legacy-1","agent":"Codex","pid":10,"started_at":"2026-08-24T00:01:00Z","last_seen_at":"2026-08-24T00:10:00Z","active":true},
      {"id":"legacy-2","agent":"Codex","pid":20,"started_at":"2026-08-24T00:02:00Z","last_seen_at":"2026-08-24T00:10:00Z","active":true},
      {"id":"legacy-ended","agent":"Codex","pid":99,"started_at":"2026-08-23T23:00:00Z","last_seen_at":"2026-08-23T23:10:00Z","ended_at":"2026-08-23T23:10:00Z","active":false}
    ]
  }]
}`
	if err := os.WriteFile(path, []byte(v1), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := openCatalog(path, logicalTestDetector)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 0, 11, 0, 0, time.UTC)
	live := ExecutionSession{
		Adapter:        "codex",
		ID:             "execution-specview",
		Agent:          "Codex",
		CWD:            "/work/specview",
		RepositoryRoot: "/work/specview",
		WorktreeRoot:   "/work/specview",
		ProcessIDs:     []int{10, 20, 30},
	}
	if _, err := catalog.ObserveExecutions([]ExecutionSession{live}, now); err != nil {
		t.Fatal(err)
	}
	sessions := catalog.Repositories()[0].Sessions
	if len(sessions) != 2 {
		t.Fatalf("sessions = %#v, want logical session plus ended legacy history", sessions)
	}
	logicalIndex := findSessionByID(sessions, live.ID)
	endedIndex := findSessionByID(sessions, "legacy-ended")
	if logicalIndex < 0 || endedIndex < 0 {
		t.Fatalf("unexpected cutover sessions: %#v", sessions)
	}
	logical := sessions[logicalIndex]
	wantStart := time.Date(2026, 8, 24, 0, 1, 0, 0, time.UTC)
	if !logical.StartedAt.Equal(wantStart) || logical.IdentityKind != SessionIdentityLogical {
		t.Fatalf("logical cutover did not preserve earliest start: %#v", logical)
	}
	if sessions[endedIndex].IdentityKind != SessionIdentityLegacyPID || sessions[endedIndex].Active {
		t.Fatalf("ended legacy history was rewritten: %#v", sessions[endedIndex])
	}
}

func TestCatalogLogicalSessionEndUsesLogicalID(t *testing.T) {
	catalog, err := openCatalog(filepath.Join(t.TempDir(), "catalog.json"), logicalTestDetector)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	first := ExecutionSession{Adapter: "codex", ID: "execution-a", Agent: "Codex", RepositoryRoot: "/work/specview", ProcessIDs: []int{10}}
	second := ExecutionSession{Adapter: "codex", ID: "execution-b", Agent: "Codex", RepositoryRoot: "/work/specview", ProcessIDs: []int{20}}
	if _, err := catalog.ObserveExecutions([]ExecutionSession{first, second}, started); err != nil {
		t.Fatal(err)
	}
	endedAt := started.Add(2 * time.Second)
	second.ProcessIDs = []int{10, 20}
	if _, err := catalog.ObserveExecutions([]ExecutionSession{second}, endedAt); err != nil {
		t.Fatal(err)
	}
	sessions := catalog.Repositories()[0].Sessions
	a := sessions[findSessionByID(sessions, first.ID)]
	b := sessions[findSessionByID(sessions, second.ID)]
	if a.Active || a.EndedAt == nil || !a.EndedAt.Equal(endedAt) {
		t.Fatalf("missing logical session end: %#v", a)
	}
	if !b.Active || b.EndedAt != nil {
		t.Fatalf("active logical session was ended by overlapping process diagnostics: %#v", b)
	}
}
