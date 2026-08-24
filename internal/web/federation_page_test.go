package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
	"github.com/sergii/specview/internal/hoststate"
)

type federationPageReaderStub struct {
	projection federationruntime.Projection
	err        error
}

func (s federationPageReaderStub) Build(context.Context) (federationruntime.Projection, error) {
	return s.projection, s.err
}

func TestFederationPagePreservesHostFreshnessAndSourceAttribution(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{
			{Source: federationruntime.HostSourceLocal, HostID: "host:550e8400-e29b-41d4-a716-446655440000", Hostname: "laptop", HasSnapshot: true},
			{Source: federationruntime.HostSourcePeer, Peer: "devbox", HostID: "host:550e8400-e29b-41d4-a716-446655440001", Hostname: "devbox", Freshness: federationpeers.FreshnessUnreachable, HasSnapshot: true, LastError: "connection refused"},
			{Source: federationruntime.HostSourcePeer, Peer: "newbox", HostID: "host:550e8400-e29b-41d4-a716-446655440002", Freshness: federationpeers.FreshnessNeverRetrieved, HasSnapshot: false},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:fixture",
				Name:    "sergii/specview",
				Active:  true,
				Instances: []federation.SourcedInstance{
					{HostID: "host:550e8400-e29b-41d4-a716-446655440000", Hostname: "laptop", RepositoryInstance: federation.RepositoryInstance{Root: "/work/laptop/specview", Active: true}},
					{HostID: "host:550e8400-e29b-41d4-a716-446655440001", Hostname: "devbox", RepositoryInstance: federation.RepositoryInstance{Root: "/work/devbox/specview", Active: false}},
				},
			}},
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/federation", nil)
	server.federationPage(federationPageReaderStub{projection: projection})(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`data-host-source="local"`,
		`data-freshness="unreachable"`,
		`connection refused`,
		`data-freshness="never_retrieved"`,
		`no snapshot yet`,
		`data-group-id="group:fixture"`,
		`data-host-id="host:550e8400-e29b-41d4-a716-446655440000"`,
		`data-host-id="host:550e8400-e29b-41d4-a716-446655440001"`,
		`/work/devbox/specview`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("federation page missing %q; body=%s", want, body)
		}
	}
}

func TestFederationPageFailsExplicitlyWithoutInventedFacts(t *testing.T) {
	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	server := NewHostServer(catalog, NewHub(), "127.0.0.1", 7331)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/federation", nil)
	server.federationPage(federationPageReaderStub{err: errors.New("projection broken")})(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "projection broken") {
		t.Fatalf("explicit projection error missing: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "sergii/specview") {
		t.Fatalf("failure response invented repository facts: %s", recorder.Body.String())
	}
}
