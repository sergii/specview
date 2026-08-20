package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sergii/specview/internal/config"
)

func TestDiscoverConfigRootsFindsImmediateConfiguredChildren(t *testing.T) {
	root := t.TempDir()
	configured := []string{"alpha", "beta"}
	for _, name := range configured {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "ignored", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored", "nested", config.FileName), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := discoverConfigRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "alpha"), filepath.Join(root, "beta")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %#v, want %#v", got, want)
	}
}

func TestDiscoverConfigRootsUsesConfiguredDirectoryAsProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, config.FileName), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := discoverConfigRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{root}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roots = %#v, want %#v", got, want)
	}
}
