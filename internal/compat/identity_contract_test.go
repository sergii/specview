package compat_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/identity"
)

func TestHostIdentityV1ContractFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")
	copyFixture(t, "host/v1.json", path)

	value, err := identity.LoadHost(path)
	if err != nil {
		t.Fatal(err)
	}
	if value.Version != identity.HostIdentityVersion {
		t.Fatalf("version = %d", value.Version)
	}
	if value.ID != "host:550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("id = %q", value.ID)
	}
	if got := value.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"); got != "2026-08-23T19:00:00Z" {
		t.Fatalf("created_at = %q", got)
	}
}

func TestConfigV1ExplicitProjectIdentityFixture(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, "config/v1-project-id.yaml", filepath.Join(root, config.FileName))

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 1 || cfg.Project.ID != "specview:sergii/specview" {
		t.Fatalf("unexpected project identity contract: %#v", cfg.Project)
	}
	if cfg.Project.Name != "Specview" || cfg.Project.Root != "." {
		t.Fatalf("unexpected project metadata: %#v", cfg.Project)
	}
}

func TestRepositoryIdentityV1CorrelationFixture(t *testing.T) {
	var fixture struct {
		SchemaVersion      int `json:"schema_version"`
		RepositoryInstance struct {
			HostID     string `json:"host_id"`
			Root       string `json:"root"`
			ExpectedID string `json:"expected_id"`
		} `json:"repository_instance"`
		Normalization struct {
			RepositoryName struct {
				Input    string `json:"input"`
				Expected string `json:"expected"`
			} `json:"repository_name"`
			GitRemotes        []string `json:"git_remotes"`
			ExpectedGitRemote string   `json:"expected_git_remote"`
		} `json:"normalization"`
		Cases []struct {
			Name     string                         `json:"name"`
			Left     identity.RepositoryFingerprint `json:"left"`
			Right    identity.RepositoryFingerprint `json:"right"`
			Expected identity.CorrelationOutcome    `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(readFixture(t, "repository-identity/v1-correlation.json"), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d", fixture.SchemaVersion)
	}

	instanceID, err := identity.RepositoryInstanceID(fixture.RepositoryInstance.HostID, fixture.RepositoryInstance.Root)
	if err != nil {
		t.Fatal(err)
	}
	if instanceID != fixture.RepositoryInstance.ExpectedID {
		t.Fatalf("RepositoryInstance ID = %q, want %q", instanceID, fixture.RepositoryInstance.ExpectedID)
	}
	if got := identity.NormalizeRepositoryName(fixture.Normalization.RepositoryName.Input); got != fixture.Normalization.RepositoryName.Expected {
		t.Fatalf("normalized repository name = %q, want %q", got, fixture.Normalization.RepositoryName.Expected)
	}
	for _, remote := range fixture.Normalization.GitRemotes {
		if got := identity.NormalizeGitRemote(remote); got != fixture.Normalization.ExpectedGitRemote {
			t.Fatalf("normalized Git remote %q = %q, want %q", remote, got, fixture.Normalization.ExpectedGitRemote)
		}
	}
	for _, tc := range fixture.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := identity.CorrelateRepositories(tc.Left, tc.Right)
			if result.Outcome != tc.Expected {
				t.Fatalf("outcome = %q, want %q; reasons=%#v", result.Outcome, tc.Expected, result.Reasons)
			}
		})
	}
}
