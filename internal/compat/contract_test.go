package compat_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/evidence"
	"github.com/sergii/specview/internal/hoststate"
)

func TestConfigV1ContractFixture(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "config/v1-acceptance.yaml", filepath.Join(root, config.FileName))

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config contract fixture: %v", err)
	}
	if cfg.Version != 1 || cfg.Project.Name != "Contract Demo" || cfg.Project.Root != "." {
		t.Fatalf("unexpected project contract: %#v", cfg.Project)
	}
	if cfg.Specs.Adapter != config.AdapterSpecview || cfg.Specs.Path != "specs" || cfg.Specs.Pattern != "*.md" {
		t.Fatalf("unexpected specs contract: %#v", cfg.Specs)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 7331 {
		t.Fatalf("unexpected server contract: %#v", cfg.Server)
	}

	policy := cfg.AcceptancePolicy()
	if len(policy.Required) != 3 {
		t.Fatalf("acceptance requirements = %d, want 3", len(policy.Required))
	}
	if policy.Required[0].Check != "unit-tests" || policy.Required[0].AllowSkipped {
		t.Fatalf("unexpected unit-tests requirement: %#v", policy.Required[0])
	}
	if policy.Required[2].Check != "hardware-in-loop" || !policy.Required[2].AllowSkipped {
		t.Fatalf("unexpected hardware-in-loop requirement: %#v", policy.Required[2])
	}
}

func TestEvidenceV1ContractFixture(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "evidence/v1-passed.json", filepath.Join(root, "record.json"))

	records, err := evidence.NewNativeAdapter(root).Scan()
	if err != nil {
		t.Fatalf("scan evidence contract fixture: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("evidence records = %d, want 1", len(records))
	}
	record := records[0]
	if record.Error != "" {
		t.Fatalf("evidence contract fixture invalid: %s", record.Error)
	}
	if record.Version != 1 || record.ID != "WORK-001-tests-20260823T120000Z" || record.WorkItemID != "WORK-001" {
		t.Fatalf("unexpected evidence identity contract: %#v", record)
	}
	if record.Revision != "git:abcdef1234567890" || record.Check != "unit-tests" || record.Kind != evidence.KindTest {
		t.Fatalf("unexpected evidence subject contract: %#v", record)
	}
	if record.Provider != "rspec" || record.Result != evidence.ResultPassed {
		t.Fatalf("unexpected evidence result contract: %#v", record)
	}
	if record.Metrics["examples"] != 42 || record.Metrics["failures"] != 0 {
		t.Fatalf("unexpected evidence metrics contract: %#v", record.Metrics)
	}
}

func TestCatalogV1ContractFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	copyFixture(t, "catalog/v1.json", path)

	catalog, err := hoststate.OpenCatalog(path)
	if err != nil {
		t.Fatalf("open catalog contract fixture: %v", err)
	}
	repositories := catalog.Repositories()
	if len(repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(repositories))
	}
	repository := repositories[0]
	if repository.ID != "repo-contract-v1" || repository.Name != "sergii/specview" || repository.Root != "/work/sergii/specview" {
		t.Fatalf("unexpected repository contract: %#v", repository)
	}
	if repository.Convention.Adapter != config.AdapterSpecview || !repository.Convention.Recognized || !repository.Convention.Supported {
		t.Fatalf("unexpected convention contract: %#v", repository.Convention)
	}
	if len(repository.Sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(repository.Sessions))
	}
	session := repository.Sessions[0]
	if session.ID != "session-contract-v1" || session.IdentityKind != hoststate.SessionIdentityLegacyPID || session.Adapter != "codex" || session.Agent != "Codex" || len(session.ProcessIDs) != 1 || session.ProcessIDs[0] != 4242 || !session.Active {
		t.Fatalf("unexpected migrated v1 session contract: %#v", session)
	}
}

func TestMCPV1ToolContractFixture(t *testing.T) {
	data := readFixture(t, "mcp/v1-tools.json")
	var fixture struct {
		SchemaVersion   int    `json:"schema_version"`
		ProtocolVersion string `json:"protocol_version"`
		Tools           []struct {
			Name      string   `json:"name"`
			Arguments []string `json:"arguments"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode MCP tool contract fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.ProtocolVersion != "2025-11-25" {
		t.Fatalf("unexpected MCP contract metadata: %#v", fixture)
	}
	wantNames := []string{
		"list_repositories",
		"get_repository",
		"list_active_sessions",
		"list_worktrees",
		"list_work_items",
		"get_work_item",
		"get_evidence",
		"get_acceptance",
	}
	if len(fixture.Tools) != len(wantNames) {
		t.Fatalf("MCP tools = %d, want %d", len(fixture.Tools), len(wantNames))
	}
	for i, want := range wantNames {
		if fixture.Tools[i].Name != want {
			t.Fatalf("tool %d = %q, want %q", i, fixture.Tools[i].Name, want)
		}
	}
	if arguments := fixture.Tools[4].Arguments; len(arguments) != 1 || arguments[0] != "repository_id" {
		t.Fatalf("unexpected list_work_items arguments: %#v", arguments)
	}
	for _, index := range []int{5, 6, 7} {
		arguments := fixture.Tools[index].Arguments
		if len(arguments) != 2 || arguments[0] != "repository_id" || arguments[1] != "work_item_id" {
			t.Fatalf("unexpected work-item arguments for %s: %#v", fixture.Tools[index].Name, arguments)
		}
	}
}

func TestMCPV1WorkItemListContractFixture(t *testing.T) {
	var result controlplane.ListWorkItemsResult
	decodeFixture(t, "mcp/v1-list-work-items.json", &result)
	if result.SchemaVersion != controlplane.SchemaVersion || result.RepositoryID != "repo-contract-v1" {
		t.Fatalf("unexpected WorkItem list metadata: %#v", result)
	}
	if len(result.WorkItems) != 1 || result.WorkItems[0].WorkItemID != "H18" {
		t.Fatalf("unexpected WorkItem list: %#v", result.WorkItems)
	}
	if result.WorkItems[0].Title != "H18 - MCP Server" || result.WorkItems[0].Status != "in_progress" {
		t.Fatalf("unexpected WorkItem discovery semantics: %#v", result.WorkItems[0])
	}
}

func TestMCPV1WorkItemContractFixture(t *testing.T) {
	var result controlplane.GetWorkItemResult
	decodeFixture(t, "mcp/v1-get-work-item.json", &result)
	if result.SchemaVersion != controlplane.SchemaVersion || result.RepositoryID != "repo-contract-v1" {
		t.Fatalf("unexpected WorkItem result metadata: %#v", result)
	}
	if result.WorkItem.WorkItemID != "H18" || result.WorkItem.Kind != "spec" || result.WorkItem.Status != "in_progress" {
		t.Fatalf("unexpected WorkItem contract: %#v", result.WorkItem)
	}
	if len(result.WorkItem.Relations) != 1 || result.WorkItem.Relations[0].Target != "H17" {
		t.Fatalf("unexpected WorkItem relations: %#v", result.WorkItem.Relations)
	}
}

func TestMCPV1EvidenceContractFixture(t *testing.T) {
	var result controlplane.GetEvidenceResult
	decodeFixture(t, "mcp/v1-get-evidence.json", &result)
	if result.SchemaVersion != controlplane.SchemaVersion || result.WorkItemID != "H18" || len(result.Records) != 1 {
		t.Fatalf("unexpected Evidence result: %#v", result)
	}
	record := result.Records[0]
	if record.ID != "H18-tests-20260823T180000Z" || record.Revision != "git:abcdef1234567890" {
		t.Fatalf("unexpected Evidence identity: %#v", record)
	}
	if record.Check != "unit-tests" || record.Provider != "go-test" || record.Result != "passed" {
		t.Fatalf("unexpected Evidence semantics: %#v", record)
	}
}

func TestMCPV1AcceptanceContractFixture(t *testing.T) {
	var result controlplane.GetAcceptanceResult
	decodeFixture(t, "mcp/v1-get-acceptance.json", &result)
	if result.SchemaVersion != controlplane.SchemaVersion || result.WorkItemID != "H18" {
		t.Fatalf("unexpected Acceptance result metadata: %#v", result)
	}
	if !result.Revision.Available || result.Revision.Revision != "git:abcdef1234567890" {
		t.Fatalf("unexpected Acceptance revision: %#v", result.Revision)
	}
	if result.Decision.State != "accepted" || len(result.Decision.Checks) != 1 {
		t.Fatalf("unexpected Acceptance decision: %#v", result.Decision)
	}
	if result.Decision.Checks[0].EvidenceID != "H18-tests-20260823T180000Z" {
		t.Fatalf("unexpected Acceptance evidence link: %#v", result.Decision.Checks[0])
	}
}

func decodeFixture(t *testing.T, relative string, destination any) {
	t.Helper()
	if err := json.Unmarshal(readFixture(t, relative), destination); err != nil {
		t.Fatalf("decode contract fixture %s: %v", relative, err)
	}
}

func readFixture(t *testing.T, relative string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), "testdata", "contracts", relative))
	if err != nil {
		t.Fatalf("read contract fixture %s: %v", relative, err)
	}
	return data
}

func copyFixture(t *testing.T, relative, destination string) {
	t.Helper()
	data := readFixture(t, relative)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("create fixture destination: %v", err)
	}
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		t.Fatalf("write fixture destination: %v", err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
