package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/web"
)

const fixtureHead = "abc123"

type fixtureSourceControl struct {
	root string
}

func (s fixtureSourceControl) Inspect(context.Context, string) (sourcecontrol.RepositoryContext, error) {
	return sourcecontrol.RepositoryContext{
		Git: sourcecontrol.GitContext{
			Remote: "https://github.com/sergii/specview.git",
			Worktrees: []sourcecontrol.Worktree{{
				Path:       s.root,
				Branch:     "feat/acceptance-policy",
				Head:       fixtureHead,
				DirtyCount: 0,
			}},
		},
	}, nil
}

type fixtureFederationReader struct {
	root string
}

func (r fixtureFederationReader) Build(context.Context) (federationruntime.Projection, error) {
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	return federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{
			{
				Source:      federationruntime.HostSourceLocal,
				HostID:      "host:550e8400-e29b-41d4-a716-446655440000",
				Hostname:    "e2e-laptop",
				HasSnapshot: true,
				ControlPlane: &controlplane.GetHostControlPlaneResult{
					SchemaVersion: controlplane.SchemaVersion,
					Host:          "e2e-laptop",
					Intent: controlplane.HostIntentSummary{
						ManagedRepositories: 1,
						WorkItems:           1,
						InProgress:          1,
					},
					Execution: controlplane.HostExecutionSummary{
						ActiveSessions:     1,
						ActiveRepositories: 1,
					},
					Evidence: controlplane.HostEvidenceSummary{
						Total:  2,
						Passed: 2,
					},
					Acceptance: controlplane.HostAcceptanceSummary{
						ConfiguredRepositories: 1,
						Accepted:               2,
					},
					Attention: []controlplane.HostAttentionSummary{},
				},
			},
			{
				Source:      federationruntime.HostSourcePeer,
				Peer:        "devbox",
				HostID:      "host:550e8400-e29b-41d4-a716-446655440001",
				Hostname:    "e2e-devbox",
				Freshness:   federationpeers.FreshnessUnreachable,
				HasSnapshot: true,
				LastError:   "fixture transport unavailable",
				ControlPlane: &controlplane.GetHostControlPlaneResult{
					SchemaVersion: controlplane.SchemaVersion,
					Host:          "e2e-devbox",
					Execution: controlplane.HostExecutionSummary{
						ActiveSessions:     2,
						ActiveRepositories: 1,
					},
					Evidence: controlplane.HostEvidenceSummary{
						Total:                3,
						Passed:               2,
						Failed:               1,
						AffectedRepositories: 1,
					},
					Acceptance: controlplane.HostAcceptanceSummary{
						ConfiguredRepositories: 1,
						Accepted:               2,
						Blocked:                1,
					},
					Attention: []controlplane.HostAttentionSummary{{
						RepositoryID:   "repo-e2e-remote",
						RepositoryName: "sergii/specview",
						LastSeenAt:     now.Add(-time.Minute),
						Signals:        []string{"1 failed Evidence record", "1 blocked Acceptance item"},
					}},
				},
			},
			{Source: federationruntime.HostSourcePeer, Peer: "newbox", HostID: "host:550e8400-e29b-41d4-a716-446655440002", Freshness: federationpeers.FreshnessNeverRetrieved, HasSnapshot: false},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:e2e-specview",
				Name:    "sergii/specview",
				Active:  true,
				Agents:  []string{"Codex"},
				Instances: []federation.SourcedInstance{
					{HostID: "host:550e8400-e29b-41d4-a716-446655440000", Hostname: "e2e-laptop", ObservedAt: now, RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:e2e-local", SourceRepositoryID: "repo-e2e", Name: "sergii/specview", Root: r.root, Active: true}},
					{HostID: "host:550e8400-e29b-41d4-a716-446655440001", Hostname: "e2e-devbox", ObservedAt: now.Add(-time.Minute), RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:e2e-devbox", SourceRepositoryID: "repo-e2e-remote", Name: "sergii/specview", Root: "/srv/repos/sergii/specview", Active: false}},
				},
			}},
		},
	}, nil
}

func main() {
	root := filepath.Join(os.TempDir(), "specview-e2e", "repository")
	if err := os.RemoveAll(filepath.Dir(root)); err != nil {
		log.Fatal(err)
	}
	if err := writeFixture(root); err != nil {
		log.Fatal(err)
	}

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		log.Fatal(err)
	}
	endedAt := time.Date(2026, 8, 23, 11, 50, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "e2e-ended-claude",
		Adapter:        "claude-code",
		Agent:          "Claude",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{4141},
		StartedAt:      endedAt.Add(-10 * time.Minute),
	}}, endedAt); err != nil {
		log.Fatal(err)
	}
	liveAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	if _, err := catalog.ObserveExecutions([]hoststate.ExecutionSession{{
		ID:             "e2e-live-codex",
		Adapter:        "codex",
		Agent:          "Codex",
		RepositoryRoot: root,
		WorktreeRoot:   root,
		CWD:            root,
		ProcessIDs:     []int{4242, 4243},
		StartedAt:      liveAt.Add(-5 * time.Minute),
	}}, liveAt); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	server := web.NewHostServerWithSources(
		catalog,
		web.NewHub(),
		"127.0.0.1",
		7332,
		nil,
		fixtureSourceControl{root: root},
	)
	log.Printf("Specview e2e fixture server listening on http://127.0.0.1:7332")
	if err := server.ListenAndServeWithFederation(ctx, fixtureFederationReader{root: root}); err != nil {
		log.Fatal(err)
	}
}

func writeFixture(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".specview", "evidence"), 0o755); err != nil {
		return err
	}

	config := `version: 2
project:
  name: "Specview E2E"
  root: "."
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
    - lint
`
	if err := os.WriteFile(filepath.Join(root, ".specview.yaml"), []byte(config), 0o644); err != nil {
		return err
	}

	spec := `---
specview:
  status: in_progress
---
# H17 Acceptance Policy

This deterministic fixture exists only for browser conformance tests.
`
	if err := os.WriteFile(filepath.Join(root, "specs", "H17.md"), []byte(spec), 0o644); err != nil {
		return err
	}

	for index, check := range []string{"unit-tests", "lint"} {
		observed := fmt.Sprintf("2026-08-23T12:00:0%dZ", index)
		record := fmt.Sprintf(`{
  "version": 1,
  "id": "H17-%s-e2e",
  "work_item_id": "H17",
  "revision": "git:%s",
  "check": %q,
  "kind": "test",
  "provider": "e2e-fixture",
  "result": "passed",
  "finished_at": %q,
  "observed_at": %q,
  "summary": "fixture passed"
}
`, check, fixtureHead, check, observed, observed)
		if err := os.WriteFile(filepath.Join(root, ".specview", "evidence", check+".json"), []byte(record), 0o644); err != nil {
			return err
		}
	}
	return nil
}
