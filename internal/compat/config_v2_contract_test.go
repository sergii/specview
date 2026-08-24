package compat_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergii/specview/internal/config"
)

func TestConfigV2RepositoryContractFixture(t *testing.T) {
	root := t.TempDir()
	data := readFixture(t, "config/v2-repository.yaml")
	if strings.Contains(string(data), "server:") {
		t.Fatal("repository config v2 fixture must not contain Host server settings")
	}
	if err := os.WriteFile(filepath.Join(root, config.FileName), data, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config v2 contract fixture: %v", err)
	}
	if cfg.Version != 2 || cfg.Project.ID != "specview:contract/v2" || cfg.Project.Name != "Contract V2" {
		t.Fatalf("unexpected config v2 project contract: %#v", cfg.Project)
	}
	if cfg.Specs.Adapter != config.AdapterSpecview || cfg.Specs.Path != "specs" || cfg.Specs.Pattern != "*.md" {
		t.Fatalf("unexpected config v2 specs contract: %#v", cfg.Specs)
	}
	if cfg.Server.Host != "" || cfg.Server.Port != 0 {
		t.Fatalf("config v2 exposed legacy repository server settings: %#v", cfg.Server)
	}
}
