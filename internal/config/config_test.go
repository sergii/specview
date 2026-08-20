package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitAndLoad(t *testing.T) {
	root := t.TempDir()

	createdConfig, createdSpecs, err := Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if !createdConfig || !createdSpecs {
		t.Fatalf("expected config and specs directory to be created")
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Specs.Path != "specs" || cfg.Specs.Pattern != "*.md" {
		t.Fatalf("unexpected specs config: %#v", cfg.Specs)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 7331 {
		t.Fatalf("unexpected server config: %#v", cfg.Server)
	}

	if _, err := os.Stat(filepath.Join(root, "specs")); err != nil {
		t.Fatal(err)
	}
}

func TestInitDoesNotOverwriteConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, FileName)
	const custom = "version: 1\nspecs:\n  path: custom\n  pattern: '*.md'\nserver:\n  host: 127.0.0.1\n  port: 8000\n"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	createdConfig, _, err := Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if createdConfig {
		t.Fatal("config should not be overwritten")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != custom {
		t.Fatal("existing config changed")
	}
}
