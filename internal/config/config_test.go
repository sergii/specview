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
	if cfg.Specs.Adapter != "specview" || cfg.Specs.Path != "specs" || cfg.Specs.Pattern != "*.md" {
		t.Fatalf("unexpected specs config: %#v", cfg.Specs)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 7331 {
		t.Fatalf("unexpected server config: %#v", cfg.Server)
	}
	if cfg.Project.Name != "" || cfg.Project.Root != "." {
		t.Fatalf("unexpected project config: %#v", cfg.Project)
	}
	if got := cfg.ResolveProjectRoot(root); got != filepath.Clean(root) {
		t.Fatalf("unexpected resolved project root: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "specs")); err != nil {
		t.Fatal(err)
	}
}

func TestInitDetectsGitHubSpecKit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".specify", "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "specs", "001-feature"), 0o755); err != nil {
		t.Fatal(err)
	}

	createdConfig, _, err := Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if !createdConfig {
		t.Fatal("expected config to be created")
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Specs.Adapter != "github-spec-kit" {
		t.Fatalf("specs.adapter = %q, want github-spec-kit", cfg.Specs.Adapter)
	}
}

func TestLoadProjectMetadata(t *testing.T) {
	root := t.TempDir()
	data := "version: 1\nproject:\n  name: \"Observed Project\"\n  root: ./demo\nspecs:\n  adapter: specview\n  path: specs\n  pattern: '*.md'\nserver:\n  host: 127.0.0.1\n  port: 7331\n"
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project.Name != "Observed Project" || cfg.Project.Root != "./demo" {
		t.Fatalf("unexpected project config: %#v", cfg.Project)
	}
	if cfg.Specs.Adapter != "specview" {
		t.Fatalf("unexpected specs adapter %q", cfg.Specs.Adapter)
	}
	want := filepath.Join(root, "demo")
	if got := cfg.ResolveProjectRoot(root); got != want {
		t.Fatalf("ResolveProjectRoot() = %q, want %q", got, want)
	}
}

func TestLoadConfigWithoutAdapterDefaultsToSpecview(t *testing.T) {
	root := t.TempDir()
	data := "version: 1\nproject:\n  name: \"Legacy\"\nspecs:\n  path: specs\n  pattern: '*.md'\nserver:\n  host: 127.0.0.1\n  port: 7331\n"
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Specs.Adapter != "specview" {
		t.Fatalf("config specs.adapter = %q, want specview", cfg.Specs.Adapter)
	}
}

func TestLoadConfigWithoutProjectRootDefaultsToCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	data := "version: 1\nproject:\n  name: \"Legacy\"\nspecs:\n  path: specs\n  pattern: '*.md'\nserver:\n  host: 127.0.0.1\n  port: 7331\n"
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project.Root != "." {
		t.Fatalf("config project.root = %q, want .", cfg.Project.Root)
	}
}

func TestResolveAbsoluteProjectRoot(t *testing.T) {
	configRoot := t.TempDir()
	projectRoot := t.TempDir()
	cfg := Config{Project: Project{Root: projectRoot}}
	if got := cfg.ResolveProjectRoot(configRoot); got != filepath.Clean(projectRoot) {
		t.Fatalf("ResolveProjectRoot() = %q, want %q", got, projectRoot)
	}
}

func TestUnknownProjectDemoKeyIsRejected(t *testing.T) {
	root := t.TempDir()
	data := "version: 1\nproject:\n  name: \"Demo\"\n  demo: true\nspecs:\n  path: specs\n  pattern: '*.md'\nserver:\n  host: 127.0.0.1\n  port: 7331\n"
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected project.demo to be rejected")
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
