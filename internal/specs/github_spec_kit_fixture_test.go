package specs

import (
	"path/filepath"
	"testing"
)

func TestGitHubSpecKitFixtureEndToEnd(t *testing.T) {
	projectRoot := filepath.Join("testdata", "github-spec-kit")
	specRoot := filepath.Join(projectRoot, "specs")

	adapter, err := NewAdapter(GitHubSpecKitAdapterName, specRoot, "*.md")
	if err != nil {
		t.Fatal(err)
	}
	store := NewStoreWithAdapter(adapter)
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}

	if store.AdapterName() != GitHubSpecKitAdapterName {
		t.Fatalf("adapter = %q, want %q", store.AdapterName(), GitHubSpecKitAdapterName)
	}

	artifacts := store.All()
	feature, ok := findArtifact(artifacts, "001-candidate-feedback")
	if !ok {
		t.Fatalf("feature artifact not found: %#v", artifacts)
	}
	if feature.Kind != ArtifactSpec {
		t.Fatalf("feature kind = %q, want %q", feature.Kind, ArtifactSpec)
	}
	if feature.Title != "Feature Specification: Candidate Feedback" {
		t.Fatalf("feature title = %q", feature.Title)
	}
	if feature.Status != StatusInProgress {
		t.Fatalf("feature status = %q, want %q", feature.Status, StatusInProgress)
	}

	assertArtifactKind(t, artifacts, "constitution", ArtifactPolicy)
	assertArtifactKind(t, artifacts, "001-candidate-feedback:plan", ArtifactPlan)
	assertArtifactKind(t, artifacts, "001-candidate-feedback:research", ArtifactResearch)
	assertArtifactKind(t, artifacts, "001-candidate-feedback:tasks", ArtifactTask)
	assertArtifactKind(t, artifacts, "001-candidate-feedback:contract:contracts/api.yaml", ArtifactContract)

	plan, ok := findArtifact(artifacts, "001-candidate-feedback:plan")
	if !ok || len(plan.Relations) != 1 {
		t.Fatalf("expected plan relation, got %#v", plan.Relations)
	}
	if plan.Relations[0].Type != "belongs_to" || plan.Relations[0].Target != "001-candidate-feedback" {
		t.Fatalf("unexpected plan relation: %#v", plan.Relations[0])
	}
}
