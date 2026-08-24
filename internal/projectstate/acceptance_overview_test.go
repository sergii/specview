package projectstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sergii/specview/internal/acceptance"
	"github.com/sergii/specview/internal/revision"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func TestAcceptanceOverviewAggregatesRepositoryWorkItems(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root)
	second := `---
specview:
  status: new
---

# H19 Waiting

No evidence yet.
`
	if err := os.WriteFile(filepath.Join(root, "specs", "H19.md"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitFixture(t, root)

	project, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	git, err := sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	revisionID := "git:" + git.Worktrees[0].Head
	writeEvidenceFixture(t, root, revisionID)

	overview, err := project.AcceptanceOverview(git)
	if err != nil {
		t.Fatal(err)
	}
	if !overview.Configured || overview.Revision.Revision != revisionID {
		t.Fatalf("unexpected overview authority: %#v", overview)
	}
	if overview.Accepted != 1 || overview.Waiting != 1 || overview.Blocked != 0 || overview.EvidenceCount != 1 {
		t.Fatalf("unexpected overview counts: %#v", overview)
	}
	if len(overview.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(overview.Items))
	}
	states := make(map[string]acceptance.State)
	for _, item := range overview.Items {
		states[item.WorkItemID] = item.State
	}
	if states["H18"] != acceptance.StateAccepted || states["H19"] != acceptance.StateWaiting {
		t.Fatalf("unexpected item states: %#v", states)
	}
}

func TestAcceptanceOverviewDirtyWorktreeFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeProjectFixture(t, root)
	initGitFixture(t, root)

	project, err := Resolve(root)
	if err != nil {
		t.Fatal(err)
	}
	cleanGit, err := sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}
	writeEvidenceFixture(t, root, "git:"+cleanGit.Worktrees[0].Head)
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirtyGit, err := sourcecontrol.InspectGit(root)
	if err != nil {
		t.Fatal(err)
	}

	overview, err := project.AcceptanceOverview(dirtyGit)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Revision.Reason != revision.ReasonDirtyWorktree || !overview.EvaluationPending {
		t.Fatalf("dirty overview must fail closed: %#v", overview)
	}
	if overview.Waiting != 1 || overview.Accepted != 0 || overview.Items[0].State != acceptance.StateWaiting || !overview.Items[0].EvaluationPending {
		t.Fatalf("unexpected dirty overview: %#v", overview)
	}
}
