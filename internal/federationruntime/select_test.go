package federationruntime

import (
	"testing"
	"time"

	"github.com/sergii/specview/internal/federation"
)

func TestExactFederationSelectorsPreserveProjectionOrderAndHostScope(t *testing.T) {
	now := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	projection := Projection{
		SchemaVersion: ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []HostStatus{
			{Source: HostSourceLocal, HostID: "host:a", Hostname: "laptop", HasSnapshot: true},
			{Source: HostSourcePeer, HostID: "host:b", Hostname: "devbox", HasSnapshot: true},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{
				{
					GroupID: "group:first",
					Name:    "sergii/specview",
					Instances: []federation.SourcedInstance{
						{HostID: "host:a", RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:a"}},
						{HostID: "host:b", RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:b-first"}},
					},
				},
				{
					GroupID: "group:second",
					Name:    "sergii/other",
					Instances: []federation.SourcedInstance{
						{HostID: "host:b", RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:b-second"}},
					},
				},
			},
		},
	}

	host, ok := SelectHost(projection, "host:b")
	if !ok || host.Hostname != "devbox" {
		t.Fatalf("SelectHost = %#v, %v", host, ok)
	}
	if _, ok := SelectHost(projection, "host"); ok {
		t.Fatal("prefix Host ID must not match")
	}

	repositories := RepositoriesForHost(projection, "host:b")
	if len(repositories) != 2 {
		t.Fatalf("RepositoriesForHost = %d, want 2", len(repositories))
	}
	if repositories[0].Group.GroupID != "group:first" || repositories[0].Instance.InstanceID != "instance:b-first" {
		t.Fatalf("first repository selection = %#v", repositories[0])
	}
	if repositories[1].Group.GroupID != "group:second" || repositories[1].Instance.InstanceID != "instance:b-second" {
		t.Fatalf("second repository selection = %#v", repositories[1])
	}

	selectedHost, repository, ok := SelectRepository(projection, "host:b", "instance:b-second")
	if !ok || selectedHost.HostID != "host:b" || repository.Group.GroupID != "group:second" || repository.Instance.InstanceID != "instance:b-second" {
		t.Fatalf("SelectRepository = host=%#v repository=%#v ok=%v", selectedHost, repository, ok)
	}
	if _, _, ok := SelectRepository(projection, "host:a", "instance:b-second"); ok {
		t.Fatal("repository instance must not match under the wrong Host")
	}
	if _, _, ok := SelectRepository(projection, "host:b", "instance:b"); ok {
		t.Fatal("RepositoryInstance prefix must not match")
	}
}
