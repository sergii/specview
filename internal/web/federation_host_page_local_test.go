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
	"github.com/sergii/specview/internal/federationruntime"
	"github.com/sergii/specview/internal/hoststate"
)

func TestFederationHostPagePreservesLocalSourceAndRepositoryAuthority(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	now := time.Date(2026, 8, 25, 7, 0, 0, 0, time.UTC)
	hostID := "host:550e8400-e29b-41d4-a716-446655440000"
	reader := &federationHostReaderStub{projection: federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{{
			Source:      federationruntime.HostSourceLocal,
			HostID:      hostID,
			Hostname:    "laptop",
			HasSnapshot: true,
			ObservedAt:  &now,
			ControlPlane: &controlplane.GetHostControlPlaneResult{
				SchemaVersion: controlplane.SchemaVersion,
				Host:          "laptop",
				Attention:     []controlplane.HostAttentionSummary{},
			},
		}},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:local",
				Name:    "sergii/specview",
				Instances: []federation.SourcedInstance{{
					HostID:     hostID,
					Hostname:   "laptop",
					ObservedAt: now,
					RepositoryInstance: federation.RepositoryInstance{
						InstanceID:         "instance:local",
						SourceRepositoryID: "repo-local",
						Name:               "sergii/specview",
						Root:               "/work/specview",
						Active:             true,
					},
				}},
			}},
		},
	}}

	recorder := httptest.NewRecorder()
	server.federationHostPage(reader)(recorder, httptest.NewRequest(http.MethodGet, "/federation/host?host="+url.QueryEscape(hostID), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-host-source="local"`,
		`data-host-control-plane="available"`,
		`>local</div>`,
		`data-instance-id="instance:local"`,
		`data-source-repository-id="repo-local"`,
		`/work/specview`,
		`href="/federation/repository?host=host%3A550e8400-e29b-41d4-a716-446655440000&amp;instance=instance%3Alocal"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("local federation Host page missing %q; body=%s", want, body)
		}
	}
	if strings.Contains(body, "peer ") {
		t.Fatalf("local Host page invented peer attribution: %s", body)
	}
}
