package specs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubSpecKitAdapterScansFeatureArtifacts(t *testing.T) {
	projectRoot := t.TempDir()
	specRoot := filepath.Join(projectRoot, "specs")
	featureRoot := filepath.Join(specRoot, "001-candidate-management")
	if err := os.MkdirAll(filepath.Join(featureRoot, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".specify", "memory"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(projectRoot, ".specify", "memory", "constitution.md"), "# Constitution\n")
	writeTestFile(t, filepath.Join(featureRoot, "spec.md"), "# Candidate management\n\nRequirements.\n")
	writeTestFile(t, filepath.Join(featureRoot, "plan.md"), "# Implementation Plan\n")
	writeTestFile(t, filepath.Join(featureRoot, "research.md"), "# Research\n")
	writeTestFile(t, filepath.Join(featureRoot, "data-model.md"), "# Data model\n")
	writeTestFile(t, filepath.Join(featureRoot, "quickstart.md"), "# Quickstart\n")
	writeTestFile(t, filepath.Join(featureRoot, "tasks.md"), "# Tasks\n\n- [x] T001 Add candidate model\n- [X] T002 Add tests\n")
	writeTestFile(t, filepath.Join(featureRoot, "contracts", "api.yaml"), "openapi: 3.1.0\n")

	adapter := NewGitHubSpecKitAdapter(projectRoot, specRoot)
	artifacts, err := adapter.Scan()
	if err != nil {
		t.Fatal(err)
	}

	feature, ok := findArtifact(artifacts, "001-candidate-management")
	if !ok {
		t.Fatalf("feature artifact not found: %#v", artifacts)
	}
	if feature.Kind != ArtifactSpec || feature.Status != StatusDone {
		t.Fatalf("unexpected feature artifact: %#v", feature)
	}
	if feature.Title != "Candidate management" {
		t.Fatalf("unexpected feature title %q", feature.Title)
	}

	assertArtifactKind(t, artifacts, "constitution", ArtifactPolicy)
	assertArtifactKind(t, artifacts, "001-candidate-management:plan", ArtifactPlan)
	assertArtifactKind(t, artifacts, "001-candidate-management:research", ArtifactResearch)
	assertArtifactKind(t, artifacts, "001-candidate-management:data-model", ArtifactDesign)
	assertArtifactKind(t, artifacts, "001-candidate-management:quickstart", ArtifactChecklist)
	assertArtifactKind(t, artifacts, "001-candidate-management:tasks", ArtifactTask)
	assertArtifactKind(t, artifacts, "001-candidate-management:contract:contracts/api.yaml", ArtifactContract)

	roots := adapter.WatchRoots()
	if len(roots) != 2 || roots[0] != specRoot {
		t.Fatalf("unexpected watch roots: %#v", roots)
	}
}

func TestGitHubSpecKitAdapterDerivesNewFromSpecOnly(t *testing.T) {
	projectRoot := t.TempDir()
	specRoot := filepath.Join(projectRoot, "specs")
	featureRoot := filepath.Join(specRoot, "002-interviews")
	if err := os.MkdirAll(featureRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(featureRoot, "spec.md"), "# Interviews\n")

	artifacts, err := NewGitHubSpecKitAdapter(projectRoot, specRoot).Scan()
	if err != nil {
		t.Fatal(err)
	}
	feature, ok := findArtifact(artifacts, "002-interviews")
	if !ok || feature.Status != StatusNew {
		t.Fatalf("expected new feature, got %#v", feature)
	}
}

func TestGitHubSpecKitAdapterDerivesInProgressFromPlanOrOpenTasks(t *testing.T) {
	projectRoot := t.TempDir()
	specRoot := filepath.Join(projectRoot, "specs")

	planned := filepath.Join(specRoot, "003-feedback")
	if err := os.MkdirAll(planned, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(planned, "spec.md"), "# Feedback\n")
	writeTestFile(t, filepath.Join(planned, "plan.md"), "# Plan\n")

	active := filepath.Join(specRoot, "004-offers")
	if err := os.MkdirAll(active, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(active, "spec.md"), "# Offers\n")
	writeTestFile(t, filepath.Join(active, "tasks.md"), "- [x] T001 Done\n- [ ] T002 Remaining\n")

	artifacts, err := NewGitHubSpecKitAdapter(projectRoot, specRoot).Scan()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"003-feedback", "004-offers"} {
		feature, ok := findArtifact(artifacts, id)
		if !ok || feature.Status != StatusInProgress {
			t.Fatalf("expected %s in progress, got %#v", id, feature)
		}
	}
}

func TestNewAdapterBuildsGitHubSpecKitAdapter(t *testing.T) {
	projectRoot := t.TempDir()
	specRoot := filepath.Join(projectRoot, "specs")
	adapter, err := NewAdapter(GitHubSpecKitAdapterName, specRoot, "*.md")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Name() != GitHubSpecKitAdapterName {
		t.Fatalf("adapter name = %q", adapter.Name())
	}
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findArtifact(artifacts []Artifact, id string) (Artifact, bool) {
	for _, artifact := range artifacts {
		if artifact.ID == id {
			return artifact, true
		}
	}
	return Artifact{}, false
}

func assertArtifactKind(t *testing.T, artifacts []Artifact, id string, want ArtifactKind) {
	t.Helper()
	artifact, ok := findArtifact(artifacts, id)
	if !ok {
		t.Fatalf("artifact %s not found", id)
	}
	if artifact.Kind != want {
		t.Fatalf("artifact %s kind = %q, want %q", id, artifact.Kind, want)
	}
}
