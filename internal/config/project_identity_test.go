package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadExplicitProjectIdentity(t *testing.T) {
	root := t.TempDir()
	data := `version: 1
project:
  id: specview:sergii/specview
  name: Specview
  root: .
specs:
  adapter: specview
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
	if cfg.Project.ID != "specview:sergii/specview" {
		t.Fatalf("project.id = %q", cfg.Project.ID)
	}
}

func TestProjectIdentityRejectsWhitespace(t *testing.T) {
	cfg := Config{
		Version: 1,
		Project: Project{ID: "specview:two projects", Root: "."},
		Specs:   Specs{Adapter: "specview", Path: "specs", Pattern: "*.md"},
		Server:  Server{Host: "127.0.0.1", Port: 7331},
	}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "project.id") {
		t.Fatalf("expected project.id validation error, got %v", err)
	}
}
