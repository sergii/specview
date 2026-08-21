package specs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSpecAdapterSeparatesKnowledgeHistoryFromActiveWork(t *testing.T) {
	root := filepath.Join("testdata", "openspec")
	adapter := NewOpenSpecAdapter(root)
	artifacts, err := adapter.Scan()
	if err != nil {
		t.Fatal(err)
	}

	policy, ok := findArtifact(artifacts, "openspec:config")
	if !ok {
		t.Fatalf("OpenSpec project config not found: %#v", artifacts)
	}
	if policy.Kind != ArtifactPolicy || policy.Plane != PlaneKnowledge || policy.Role != RoleSupporting {
		t.Fatalf("unexpected OpenSpec policy artifact: %#v", policy)
	}
	if policy.IsBoardItem() {
		t.Fatal("OpenSpec project config must not become an active board item")
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

	archived, ok := findArtifact(artifacts, "archive:2026-08-01-old-change")
	if !ok {
		t.Fatalf("archived change history not found: %#v", artifacts)
	}
	if archived.Kind != ArtifactProposal || archived.Plane != PlaneKnowledge || archived.Role != RoleSupporting {
		t.Fatalf("unexpected archived change artifact: %#v", archived)
	}
	if archived.WorkItemID != "archive:2026-08-01-old-change" || archived.IsBoardItem() {
		t.Fatalf("archived change must remain searchable history, not active work: %#v", archived)
	}
	if !hasRelation(archived.Relations, "archived_from", "old-change") {
		t.Fatalf("archived change does not retain original change identity: %#v", archived.Relations)
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

func TestOpenSpecAdapterIndexesArchivedDeltaSpecsAsKnowledge(t *testing.T) {
	projectRoot := t.TempDir()
	root := filepath.Join(projectRoot, "openspec")
	archiveRoot := filepath.Join(root, "changes", "archive", "2026-08-21-add-search")
	writeTestFile(t, filepath.Join(archiveRoot, "proposal.md"), "# Add search\n")
	writeTestFile(t, filepath.Join(archiveRoot, "specs", "search", "spec.md"), "# Search\n")

	artifacts, err := NewOpenSpecAdapter(root).Scan()
	if err != nil {
		t.Fatal(err)
	}
	delta, ok := findArtifact(artifacts, "archive:2026-08-21-add-search:delta:search")
	if !ok {
		t.Fatalf("archived delta spec not found: %#v", artifacts)
	}
	if delta.Plane != PlaneKnowledge || delta.Role != RoleSupporting || delta.IsBoardItem() {
		t.Fatalf("unexpected archived delta projection: %#v", delta)
	}
	if !hasRelation(delta.Relations, "changes", "current:search") || !hasRelation(delta.Relations, "archived_from", "add-search") {
		t.Fatalf("archived delta relations are incomplete: %#v", delta.Relations)
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

func TestArchivedOriginalChangeID(t *testing.T) {
	if got := archivedOriginalChangeID("2026-08-21-add-search"); got != "add-search" {
		t.Fatalf("archivedOriginalChangeID() = %q, want add-search", got)
	}
	if got := archivedOriginalChangeID("legacy-folder"); got != "legacy-folder" {
		t.Fatalf("archivedOriginalChangeID() = %q, want legacy-folder", got)
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
