package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
			`href="/project?id=repo-local"`,
			`data-session-id="session-local"`,
			`data-worktree-path="/work/specview"`,
			`h31-federation-repository-drilldown`,
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("local federation repository page missing %q; body=%s", want, body)
			}
		}
	})

	t.Run("remote instance remains snapshot only", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		query := url.Values{"host": {remoteHostID}, "instance": {"instance:remote"}}
		server.federationRepositoryPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/repository?"+query.Encode(), nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		for _, want := range []string{`data-host-source="peer"`, `data-freshness="unreachable"`, `connection refused`, `Remote last-known repository snapshot`} {
			if !strings.Contains(body, want) {
				t.Fatalf("remote federation repository page missing %q; body=%s", want, body)
			}
		}
		if strings.Contains(body, `/project?id=repo-remote`) {
			t.Fatalf("remote snapshot invented local repository navigation: %s", body)
		}
	})
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
