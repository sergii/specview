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
	if cfg.Project.Name != "" || cfg.Project.Demo {
		t.Fatalf("unexpected project config: %#v", cfg.Project)
	}
	if _, err := os.Stat(filepath.Join(root, "specs")); err != nil {
		t.Fatal(err)
	}
}

func TestLoadProjectMetadata(t *testing.T) {
	root := t.TempDir()
	data := "version: 1\nproject:\n  name: \"Demo Project\"\n  demo: true\nspecs:\n  path: specs\n  pattern: '*.md'\nserver:\n  host: 127.0.0.1\n  port: 7331\n"
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project.Name != "Demo Project" || !cfg.Project.Demo {
		t.Fatalf("unexpected project config: %#v", cfg.Project)
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
