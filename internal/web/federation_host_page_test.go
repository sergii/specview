package web

import (
	"context"
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

type federationHostReaderStub struct {
	projection federationruntime.Projection
	calls      int
}

func (s *federationHostReaderStub) Build(context.Context) (federationruntime.Projection, error) {
	s.calls++
	return s.projection, nil
}

func TestFederationHostPageRendersSourceControlPlaneAndRepositories(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	now := time.Date(2026, 8, 25, 6, 0, 0, 0, time.UTC)
	retrieved := now.Add(-20 * time.Second)
	attempted := now.Add(-5 * time.Second)
	succeeded := now.Add(-20 * time.Second)
	observed := now.Add(-30 * time.Second)
	age := int64(30)
	hostID := "host:550e8400-e29b-41d4-a716-446655440001"
	instanceID := "instance:remote"
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{{
			Source:           federationruntime.HostSourcePeer,
			Peer:             "devbox",
			HostID:           hostID,
			Hostname:         "devbox-01",
			Freshness:        federationpeers.FreshnessUnreachable,
			HasSnapshot:      true,
			ObservedAt:       &observed,
			RetrievedAt:      &retrieved,
			LastAttemptAt:    &attempted,
			LastSuccessAt:    &succeeded,
			LastError:        "connection refused",
			SourceAgeSeconds: &age,
			ControlPlane: &controlplane.GetHostControlPlaneResult{
				SchemaVersion: controlplane.SchemaVersion,
				Host:          "devbox-01",
				Intent: controlplane.HostIntentSummary{
					ManagedRepositories: 2,
					WorkItems:           7,
					New:                 1,
					InProgress:          2,
					Done:                3,
					Invalid:             1,
					Unavailable:         1,
				},
				Execution: controlplane.HostExecutionSummary{ActiveSessions: 2, ActiveRepositories: 1},
				Evidence: controlplane.HostEvidenceSummary{
					Total:                4,
					Passed:               2,
					Failed:               1,
					Invalid:              1,
					AffectedRepositories: 1,
					Unavailable:          1,
				},
				Acceptance: controlplane.HostAcceptanceSummary{
					ConfiguredRepositories:        1,
					UnconfiguredRepositories:      1,
					Accepted:                      2,
					Waiting:                       1,
					Blocked:                       1,
					Unconfigured:                  1,
					Invalid:                       1,
					EvaluationPendingRepositories: 1,
					UnavailableRepositories:       1,
				},
				Attention: []controlplane.HostAttentionSummary{{
					RepositoryID:   "repo-remote",
					RepositoryName: "sergii/specview",
					LastSeenAt:     observed,
					Signals:        []string{"1 failed Evidence record", "1 blocked Acceptance item"},
				}},
			},
		}},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:fixture",
				Name:    "sergii/specview",
				Instances: []federation.SourcedInstance{{
					HostID:     hostID,
					Hostname:   "devbox-01",
					ObservedAt: observed,
					RepositoryInstance: federation.RepositoryInstance{
						InstanceID:         instanceID,
						SourceRepositoryID: "repo-remote",
						Name:               "sergii/specview",
						Root:               "/srv/specview",
						Active:             true,
					},
				}},
			}},
		},
	}
	reader := &federationHostReaderStub{projection: projection}
	query := url.Values{"host": {hostID}}
	recorder := httptest.NewRecorder()
	server.federationHostPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/host?"+query.Encode(), nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if reader.calls != 1 {
		t.Fatalf("projection Build calls = %d, want 1", reader.calls)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-federation-host="` + hostID + `"`,
		`data-host-source="peer"`,
		`data-freshness="unreachable"`,
		`data-host-control-plane="available"`,
		`connection refused`,
		`data-facet="intent"`,
		`data-facet="execution"`,
		`data-facet="evidence"`,
		`data-facet="acceptance"`,
		`<strong>2</strong><span>active sessions</span>`,
		`<strong>1</strong><span>failed/error</span>`,
		`<strong>1</strong><span>blocked</span>`,
		`1 failed Evidence record`,
		`data-attention-repository="repo-remote"`,
		`data-instance-id="instance:remote"`,
		`data-source-repository-id="repo-remote"`,
		`/srv/specview`,
		`href="/federation/repository?host=host%3A550e8400-e29b-41d4-a716-446655440001&amp;instance=instance%3Aremote"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("federation Host page missing %q; body=%s", want, body)
		}
	}
}

func TestFederationHostPageKeepsUnavailableControlPlaneExplicit(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	hostID := "host:550e8400-e29b-41d4-a716-446655440002"
	reader := &federationHostReaderStub{projection: federationruntime.Projection{Hosts: []federationruntime.HostStatus{{
		Source:      federationruntime.HostSourcePeer,
		Peer:        "newbox",
		HostID:      hostID,
		Freshness:   federationpeers.FreshnessNeverRetrieved,
		HasSnapshot: false,
	}}}}

	recorder := httptest.NewRecorder()
	server.federationHostPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/host?host="+url.QueryEscape(hostID), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `data-host-control-plane="unavailable"`) || !strings.Contains(body, "Control plane unavailable for this Host") {
		t.Fatalf("explicit unavailable state missing: %s", body)
	}
	if strings.Contains(body, `data-facet="intent"`) || strings.Contains(body, `data-facet="acceptance"`) {
		t.Fatalf("unavailable Host invented control-plane facets: %s", body)
	}
}

func TestFederationHostPageRequiresExactKnownHost(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	known := "host:550e8400-e29b-41d4-a716-446655440000"
	reader := &federationHostReaderStub{projection: federationruntime.Projection{Hosts: []federationruntime.HostStatus{{HostID: known}}}}

	t.Run("missing", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		server.federationHostPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/host", nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", recorder.Code)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		server.federationHostPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/host?host=host%3Amissing", nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", recorder.Code)
		}
	})
}

func TestFederationHostAttentionLinksOnlyExactUniqueRepositoryInstance(t *testing.T) {
	host := federationruntime.HostStatus{ControlPlane: &controlplane.GetHostControlPlaneResult{Attention: []controlplane.HostAttentionSummary{{RepositoryID: "repo-a"}, {RepositoryID: "repo-b"}}}}
	repositories := []federationHostRepositoryData{
		{Instance: federation.SourcedInstance{RepositoryInstance: federation.RepositoryInstance{SourceRepositoryID: "repo-a"}}, Href: "/a"},
		{Instance: federation.SourcedInstance{RepositoryInstance: federation.RepositoryInstance{SourceRepositoryID: "repo-b"}}, Href: "/b1"},
		{Instance: federation.SourcedInstance{RepositoryInstance: federation.RepositoryInstance{SourceRepositoryID: "repo-b"}}, Href: "/b2"},
	}
	attention := federationHostAttention(host, repositories)
	if len(attention) != 2 {
		t.Fatalf("attention rows = %d, want 2", len(attention))
	}
	if attention[0].Href != "/a" {
		t.Fatalf("unique repository link = %q, want /a", attention[0].Href)
	}
	if attention[1].Href != "" {
		t.Fatalf("ambiguous repository invented link %q", attention[1].Href)
	}
}
