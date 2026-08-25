package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
	"github.com/sergii/specview/internal/hoststate"
)

func TestFederationRepositoryPagePreservesSourceAuthority(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	localHostID := "host:550e8400-e29b-41d4-a716-446655440000"
	remoteHostID := "host:550e8400-e29b-41d4-a716-446655440001"
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{
			{Source: federationruntime.HostSourceLocal, HostID: localHostID, Hostname: "laptop", HasSnapshot: true},
			{Source: federationruntime.HostSourcePeer, Peer: "devbox", HostID: remoteHostID, Hostname: "devbox", Freshness: federationpeers.FreshnessUnreachable, HasSnapshot: true, LastError: "connection refused"},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:fixture",
				Name:    "sergii/specview",
				Active:  true,
				Instances: []federation.SourcedInstance{
					{
						HostID:     localHostID,
						Hostname:   "laptop",
						ObservedAt: now,
						RepositoryInstance: federation.RepositoryInstance{
							InstanceID:         "instance:local",
							SourceRepositoryID: "repo-local",
							Name:               "sergii/specview",
							Root:               "/work/specview",
							Active:             true,
							Sessions:           []federation.Session{{ID: "session-local", Adapter: "codex", Agent: "Codex", CWD: "/work/specview", WorktreeRoot: "/work/specview", StartedAt: "2026-08-24T11:55:00Z"}},
							Worktrees:          []federation.Worktree{{Path: "/work/specview", Branch: "h31-federation-repository-drilldown", Head: "abc123", DirtyCount: 0}},
							ControlPlane:       repositoryControlPlaneFixture("laptop", "repo-local", "sergii/specview", 2, 1, 1),
						},
					},
					{
						HostID:     remoteHostID,
						Hostname:   "devbox",
						ObservedAt: now.Add(-time.Minute),
						RepositoryInstance: federation.RepositoryInstance{
							InstanceID:         "instance:remote",
							SourceRepositoryID: "repo-remote",
							Name:               "sergii/specview",
							Root:               "/srv/specview",
							ControlPlane:       repositoryControlPlaneFixture("devbox", "repo-remote", "sergii/specview", 5, 2, 3),
						},
					},
				},
			}},
		},
	}
	reader := federationPageReaderStub{projection: projection}

	t.Run("local instance links to live local repository", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		query := url.Values{"host": {localHostID}, "instance": {"instance:local"}}
		server.federationRepositoryPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/repository?"+query.Encode(), nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		for _, want := range []string{
			`data-federation-instance="instance:local"`,
			`data-host-source="local"`,
			`data-repository-control-plane="available"`,
			`href="/project?id=repo-local"`,
			`data-session-id="session-local"`,
			`data-worktree-path="/work/specview"`,
			`h31-federation-repository-drilldown`,
			`data-plane="intent"`,
			`2 work items`,
			`data-plane="execution"`,
			`data-plane="evidence"`,
			`data-plane="acceptance"`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("local federation repository page missing %q; body=%s", want, body)
			}
		}
	})

	t.Run("remote unreachable instance keeps captured control plane and remains snapshot only", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		query := url.Values{"host": {remoteHostID}, "instance": {"instance:remote"}}
		server.federationRepositoryPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/repository?"+query.Encode(), nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		for _, want := range []string{`data-host-source="peer"`, `data-freshness="unreachable"`, `data-repository-control-plane="available"`, `connection refused`, `Remote last-known repository snapshot`, `5 work items`, `3 accepted`} {
			if !strings.Contains(body, want) {
				t.Fatalf("remote federation repository page missing %q; body=%s", want, body)
			}
		}
		if strings.Contains(body, `/project?id=repo-remote`) {
			t.Fatalf("remote snapshot invented local repository navigation: %s", body)
		}
	})
}

func TestFederationRepositoryPageMarksOlderSnapshotControlPlaneUnavailable(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	hostID := "host:550e8400-e29b-41d4-a716-446655440001"
	reader := federationPageReaderStub{projection: federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		Hosts: []federationruntime.HostStatus{{Source: federationruntime.HostSourcePeer, Peer: "legacy", HostID: hostID, Hostname: "legacy", HasSnapshot: true}},
		Federation: federation.Projection{Repositories: []federation.RepositoryGroup{{Instances: []federation.SourcedInstance{{
			HostID: hostID, Hostname: "legacy", RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:legacy", SourceRepositoryID: "repo-legacy", Name: "legacy/repo", Root: "/srv/legacy"},
		}}}}},
	}}
	query := url.Values{"host": {hostID}, "instance": {"instance:legacy"}}
	recorder := httptest.NewRecorder()
	server.federationRepositoryPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/repository?"+query.Encode(), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{`data-repository-control-plane="unavailable"`, `Older v1/v2 peers are not interpreted as zero or healthy.`} {
		if !strings.Contains(body, want) {
			t.Fatalf("older federation repository page missing %q; body=%s", want, body)
		}
	}
}

func TestFederationRepositoryPageRejectsUnknownInstance(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	hostID := "host:550e8400-e29b-41d4-a716-446655440000"
	reader := federationPageReaderStub{projection: federationruntime.Projection{Hosts: []federationruntime.HostStatus{{Source: federationruntime.HostSourceLocal, HostID: hostID}}}}
	query := url.Values{"host": {hostID}, "instance": {"missing"}}
	recorder := httptest.NewRecorder()
	server.federationRepositoryPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/repository?"+query.Encode(), nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", recorder.Code, recorder.Body.String())
	}
}

func repositoryControlPlaneFixture(host, repositoryID, repositoryName string, total, passed, accepted int) *controlplane.GetRepositoryControlPlaneResult {
	return &controlplane.GetRepositoryControlPlaneResult{
		SchemaVersion:  controlplane.SchemaVersion,
		Host:           host,
		RepositoryID:   repositoryID,
		RepositoryName: repositoryName,
		Intent:         controlplane.RepositoryIntentSummary{Total: total, InProgress: total - 1, Done: 1},
		Evidence:       controlplane.RepositoryEvidenceOverviewSummary{Total: passed, Passed: passed},
		Acceptance:     controlplane.RepositoryAcceptanceOverviewSummary{Configured: true, Accepted: accepted},
	}
}
