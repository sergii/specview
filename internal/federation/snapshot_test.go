package federation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
)

type snapshotReaderStub struct {
	repositories controlplane.ListRepositoriesResult
	details      map[string]controlplane.GetRepositoryResult
	sessions     controlplane.ListActiveSessionsResult
	sessionErr   error
}

func (s snapshotReaderStub) ListRepositories(context.Context) (controlplane.ListRepositoriesResult, error) {
	return s.repositories, nil
}

func (s snapshotReaderStub) GetRepository(_ context.Context, repositoryID string) (controlplane.GetRepositoryResult, error) {
	return s.details[repositoryID], nil
}

func (s snapshotReaderStub) ListActiveSessions(context.Context) (controlplane.ListActiveSessionsResult, error) {
	return s.sessions, s.sessionErr
}

func TestBuilderProducesSourceAttributedSnapshot(t *testing.T) {
	root := t.TempDir()
	configBody := `version: 1
project:
  id: specview:team/app
  name: Team App
  root: .
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
server:
  host: 127.0.0.1
  port: 7331
`
	if err := os.WriteFile(filepath.Join(root, ".specview.yaml"), []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}

	repository := controlplane.RepositorySummary{
		ID:     "repo-local",
		Name:   "team/app",
		Root:   root,
		Active: true,
		Agents: []string{"Codex"},
	}
	reader := snapshotReaderStub{
		repositories: controlplane.ListRepositoriesResult{
			SchemaVersion: controlplane.SchemaVersion,
			Host:          "laptop",
			Repositories:  []controlplane.RepositorySummary{repository},
		},
		details: map[string]controlplane.GetRepositoryResult{
			"repo-local": {
				SchemaVersion: controlplane.SchemaVersion,
				Host:          "laptop",
				Repository: controlplane.RepositoryDetail{
					RepositorySummary: repository,
					Git: &controlplane.GitSummary{
						Remote: "git@github.com:team/app.git",
						Worktrees: []controlplane.WorktreeSummary{{
							Path:       root,
							Branch:     "main",
							Head:       "abcdef",
							DirtyCount: 1,
						}},
					},
					Forge: &controlplane.ForgeSummary{
						Provider:   "github",
						Matched:    true,
						Available:  true,
						Repository: "team/app",
					},
				},
			},
		},
		sessions: controlplane.ListActiveSessionsResult{
			SchemaVersion: controlplane.SchemaVersion,
			Host:          "laptop",
			Sessions: []controlplane.SessionSummary{{
				ID:             "session-1",
				Adapter:        "codex",
				Agent:          "Codex",
				RepositoryID:   "repo-local",
				RepositoryRoot: root,
				WorktreeRoot:   root,
				CWD:            root,
				StartedAt:      "2026-08-23T20:00:00Z",
			}},
		},
	}

	builder := NewBuilder("host:11111111-1111-4111-9111-111111111111", reader)
	builder.now = func() time.Time {
		return time.Date(2026, 8, 23, 20, 5, 0, 0, time.UTC)
	}
	snapshot, err := builder.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.HostID != "host:11111111-1111-4111-9111-111111111111" || snapshot.Hostname != "laptop" {
		t.Fatalf("unexpected Host snapshot: %#v", snapshot)
	}
	if len(snapshot.Instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(snapshot.Instances))
	}
	instance := snapshot.Instances[0]
	if instance.Fingerprint.ExplicitID != "specview:team/app" {
		t.Fatalf("explicit project identity missing: %#v", instance.Fingerprint)
	}
	if instance.Fingerprint.GitRemote != "git@github.com:team/app.git" {
		t.Fatalf("Git remote missing: %#v", instance.Fingerprint)
	}
	if instance.Fingerprint.ForgeProvider != "github" || instance.Fingerprint.ForgeRepository != "team/app" {
		t.Fatalf("forge identity missing: %#v", instance.Fingerprint)
	}
	if len(instance.Sessions) != 1 || instance.Sessions[0].ID != "session-1" {
		t.Fatalf("unexpected sessions: %#v", instance.Sessions)
	}
	if len(instance.Worktrees) != 1 || instance.Worktrees[0].DirtyCount != 1 {
		t.Fatalf("unexpected worktrees: %#v", instance.Worktrees)
	}
}

func TestDecodeSnapshotRejectsUnknownFields(t *testing.T) {
	data := readFederationFixture(t, "v1-laptop.json")
	body := strings.Replace(string(data), `"hostname": "sergii-macbook",`, `"hostname": "sergii-macbook", "unexpected": true,`, 1)
	if _, err := DecodeSnapshot([]byte(body)); err == nil {
		t.Fatal("expected unknown federation snapshot field to fail")
	}
}

func TestSnapshotRejectsInstanceIdentityFromAnotherHost(t *testing.T) {
	snapshot := loadSnapshotFixture(t, "v1-laptop.json")
	snapshot.Instances[0].InstanceID = "instance:18bdda6e039c54af7052414c4417a2f9"
	if err := snapshot.Validate(); err == nil {
		t.Fatal("expected RepositoryInstance bound to another Host/root to fail")
	}
}

func TestBuilderDegradesWhenLiveSessionsAreUnavailable(t *testing.T) {
	root := t.TempDir()
	repository := controlplane.RepositorySummary{ID: "repo-local", Name: "team/app", Root: root}
	reader := snapshotReaderStub{
		repositories: controlplane.ListRepositoriesResult{
			SchemaVersion: controlplane.SchemaVersion,
			Host:          "laptop",
			Repositories:  []controlplane.RepositorySummary{repository},
		},
		details: map[string]controlplane.GetRepositoryResult{
			"repo-local": {
				SchemaVersion: controlplane.SchemaVersion,
				Host:          "laptop",
				Repository:    controlplane.RepositoryDetail{RepositorySummary: repository},
			},
		},
		sessionErr: os.ErrPermission,
	}

	snapshot, err := NewBuilder("host:11111111-1111-4111-9111-111111111111", reader).Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Warnings) != 1 || !strings.Contains(snapshot.Warnings[0], "active sessions unavailable") {
		t.Fatalf("expected degraded session warning, got %#v", snapshot.Warnings)
	}
	if len(snapshot.Instances) != 1 {
		t.Fatalf("repository facts should survive session failure: %#v", snapshot.Instances)
	}
}
