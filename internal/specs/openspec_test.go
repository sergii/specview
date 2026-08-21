package specs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSpecAdapterSeparatesCurrentKnowledgeFromActiveWork(t *testing.T) {
	root := filepath.Join("testdata", "openspec")
	adapter := NewOpenSpecAdapter(root)
	artifacts, err := adapter.Scan()
	if err != nil {
		t.Fatal(err)
	}

	current, ok := findArtifact(artifacts, "current:auth")
	if !ok {
		t.Fatalf("current auth spec not found: %#v", artifacts)
	}
	if current.Kind != ArtifactSpec || current.Plane != PlaneKnowledge || current.Role != RolePrimary {
		t.Fatalf("unexpected current spec: %#v", current)
	}
	if current.IsBoardItem() {
		t.Fatal("current knowledge must not become an active board item")
	}

	change, ok := findArtifact(artifacts, "add-two-factor-auth")
	if !ok {
		t.Fatalf("active change not found: %#v", artifacts)
	}
	if change.Kind != ArtifactProposal || change.Plane != PlaneWork || change.Role != RolePrimary {
		t.Fatalf("unexpected active change: %#v", change)
	}
	if change.WorkItemID != "add-two-factor-auth" || change.Status != StatusInProgress || !change.IsBoardItem() {
		t.Fatalf("unexpected work-item projection: %#v", change)
	}

	delta, ok := findArtifact(artifacts, "add-two-factor-auth:delta:auth")
	if !ok {
		t.Fatalf("delta spec not found: %#v", artifacts)
	}
	if delta.Plane != PlaneWork || delta.Role != RoleSupporting || delta.WorkItemID != "add-two-factor-auth" {
		t.Fatalf("unexpected delta spec: %#v", delta)
	}
	if !hasRelation(delta.Relations, "changes", "current:auth") {
		t.Fatalf("delta spec does not point at current auth spec: %#v", delta.Relations)
	}

	if _, ok := findArtifact(artifacts, "2026-08-01-old-change"); ok {
		t.Fatal("archived change must not be active work")
	}
	if roots := adapter.WatchRoots(); len(roots) != 1 || roots[0] != filepath.Clean(root) {
		t.Fatalf("unexpected OpenSpec watch roots: %#v", roots)
	}
}

func TestOpenSpecAdapterDerivesNewAndDoneStatuses(t *testing.T) {
	projectRoot := t.TempDir()
	root := filepath.Join(projectRoot, "openspec")

	newChange := filepath.Join(root, "changes", "proposal-only")
	writeTestFile(t, filepath.Join(newChange, "proposal.md"), "# Proposal: Proposal Only\n")

	doneChange := filepath.Join(root, "changes", "completed-change")
	writeTestFile(t, filepath.Join(doneChange, "proposal.md"), "# Proposal: Completed Change\n")
	writeTestFile(t, filepath.Join(doneChange, "tasks.md"), "# Tasks\n- [x] 1.1 Done\n- [X] 1.2 Done too\n")

	artifacts, err := NewOpenSpecAdapter(root).Scan()
	if err != nil {
		t.Fatal(err)
	}
	proposalOnly, ok := findArtifact(artifacts, "proposal-only")
	if !ok || proposalOnly.Status != StatusNew {
		t.Fatalf("expected proposal-only change to be new, got %#v", proposalOnly)
	}
	completed, ok := findArtifact(artifacts, "completed-change")
	if !ok || completed.Status != StatusDone {
		t.Fatalf("expected completed change to be done, got %#v", completed)
	}
}

func TestOpenSpecAdapterSupportsFluidArtifactOrder(t *testing.T) {
	projectRoot := t.TempDir()
	root := filepath.Join(projectRoot, "openspec")
	changeRoot := filepath.Join(root, "changes", "design-first")
	writeTestFile(t, filepath.Join(changeRoot, "design.md"), "# Design: Design First\n")

	artifacts, err := NewOpenSpecAdapter(root).Scan()
	if err != nil {
		t.Fatal(err)
	}
	change, ok := findArtifact(artifacts, "design-first")
	if !ok {
		t.Fatalf("design-first work item not found: %#v", artifacts)
	}
	if change.Kind != ArtifactDesign || change.Role != RolePrimary || change.Status != StatusInProgress {
		t.Fatalf("unexpected design-first projection: %#v", change)
	}
}

func TestNewAdapterBuildsOpenSpecAdapter(t *testing.T) {
	projectRoot := t.TempDir()
	root := filepath.Join(projectRoot, "openspec")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(OpenSpecAdapterName, root, "*.md")
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Name() != OpenSpecAdapterName {
		t.Fatalf("adapter name = %q, want %q", adapter.Name(), OpenSpecAdapterName)
	}
}

func hasRelation(relations []Relation, relationType, target string) bool {
	for _, relation := range relations {
		if relation.Type == relationType && relation.Target == target {
			return true
		}
	}
	return false
}
