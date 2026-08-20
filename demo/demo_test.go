package demo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sergii/specview/demo"
	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/specs"
)

func TestCreateDemoProject(t *testing.T) {
	root := t.TempDir()
	created, err := demo.Create(root)
	if err != nil {
		t.Fatal(err)
	}
	if created != 10 {
		t.Fatalf("expected 10 demo specs, got %d", created)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project.Name != "Demo Project" || !cfg.Project.Demo {
		t.Fatalf("unexpected project config: %#v", cfg.Project)
	}
	store := specs.NewStore(filepath.Join(root, cfg.Specs.Path), cfg.Specs.Pattern)
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	items := store.All()
	if len(items) != 10 {
		t.Fatalf("expected 10 specs, got %d", len(items))
	}
	counts := map[specs.Status]int{}
	for _, item := range items {
		counts[item.Status]++
	}
	if counts[specs.StatusNew] != 4 || counts[specs.StatusInProgress] != 3 || counts[specs.StatusDone] != 3 {
		t.Fatalf("unexpected status distribution: %#v", counts)
	}
}

func TestCreateDoesNotOverwriteExistingProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".specview.yaml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := demo.Create(root); err == nil {
		t.Fatal("expected existing config to block demo initialization")
	}
}
