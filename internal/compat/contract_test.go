package compat_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sergii/specview/internal/config"
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
	if session.ID != "session-contract-v1" || session.Agent != "Codex" || session.PID != 4242 || !session.Active {
		t.Fatalf("unexpected session contract: %#v", session)
	}
}

func copyFixture(t *testing.T, relative, destination string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), "testdata", "contracts", relative))
	if err != nil {
		t.Fatalf("read contract fixture %s: %v", relative, err)
	}
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
