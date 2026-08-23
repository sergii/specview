package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sergii/specview/internal/acceptance"
)

func TestLoadAcceptancePolicy(t *testing.T) {
	root := t.TempDir()
	data := `version: 1
project:
  name: "Acceptance Demo"
  root: "."
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
    - lint
    - hardware-in-loop
  allow_skipped:
    - hardware-in-loop
server:
  host: 127.0.0.1
  port: 7331
`
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Acceptance.Required) != 3 {
		t.Fatalf("len(acceptance.required) = %d, want 3", len(cfg.Acceptance.Required))
	}
	if len(cfg.Acceptance.AllowSkipped) != 1 || cfg.Acceptance.AllowSkipped[0] != "hardware-in-loop" {
		t.Fatalf("unexpected acceptance.allow_skipped: %#v", cfg.Acceptance.AllowSkipped)
	}

	policy := cfg.AcceptancePolicy()
	if len(policy.Required) != 3 {
		t.Fatalf("len(policy.Required) = %d, want 3", len(policy.Required))
	}
	if policy.Required[0] != (acceptance.Requirement{Check: "unit-tests"}) {
		t.Fatalf("unexpected first requirement: %#v", policy.Required[0])
	}
	if policy.Required[2] != (acceptance.Requirement{Check: "hardware-in-loop", AllowSkipped: true}) {
		t.Fatalf("unexpected skipped requirement: %#v", policy.Required[2])
	}
}

func TestLoadWithoutAcceptanceKeepsPolicyUnconfigured(t *testing.T) {
	root := t.TempDir()
	data := `version: 1
project:
  name: "Legacy"
specs:
  path: specs
  pattern: "*.md"
server:
  host: 127.0.0.1
  port: 7331
`
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AcceptancePolicy().Required) != 0 {
		t.Fatalf("legacy config unexpectedly configured Acceptance: %#v", cfg.AcceptancePolicy())
	}
}

func TestLoadRejectsAcceptanceAllowSkippedUnlessRequired(t *testing.T) {
	root := t.TempDir()
	data := acceptanceConfig(`acceptance:
  required:
    - unit-tests
  allow_skipped:
    - hardware-in-loop
`)
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(root); err == nil {
		t.Fatal("Load() error = nil, want invalid allow_skipped error")
	}
}

func TestLoadRejectsDuplicateAcceptanceChecks(t *testing.T) {
	root := t.TempDir()
	data := acceptanceConfig(`acceptance:
  required:
    - unit-tests
    - unit-tests
`)
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(root); err == nil {
		t.Fatal("Load() error = nil, want duplicate acceptance check error")
	}
}

func TestLoadRejectsUnknownAcceptanceList(t *testing.T) {
	root := t.TempDir()
	data := acceptanceConfig(`acceptance:
  providers:
    - rspec
`)
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(root); err == nil {
		t.Fatal("Load() error = nil, want unknown acceptance key error")
	}
}

func acceptanceConfig(section string) string {
	return `version: 1
project:
  name: "Acceptance Demo"
  root: "."
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
` + section + `server:
  host: 127.0.0.1
  port: 7331
`
}
