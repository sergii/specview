package mcpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
)

func TestFederationHostToolReturnsExactHostAndOnlyItsRepositoryInstances(t *testing.T) {
	now := time.Date(2026, 8, 25, 7, 30, 0, 0, time.UTC)
	remoteHostID := "host:550e8400-e29b-41d4-a716-446655440001"
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{
			{
				Source:      federationruntime.HostSourceLocal,
				HostID:      "host:550e8400-e29b-41d4-a716-446655440000",
				Hostname:    "laptop",
				HasSnapshot: true,
			},
			{
				Source:      federationruntime.HostSourcePeer,
				Peer:        "devbox",
				HostID:      remoteHostID,
				Hostname:    "devbox",
				Freshness:   federationpeers.FreshnessUnreachable,
				HasSnapshot: true,
				LastError:   "dial failed",
				ControlPlane: &controlplane.GetHostControlPlaneResult{
					SchemaVersion: controlplane.SchemaVersion,
					Host:          "devbox",
					Execution:     controlplane.HostExecutionSummary{ActiveSessions: 2},
					Evidence:      controlplane.HostEvidenceSummary{Failed: 1},
					Acceptance:    controlplane.HostAcceptanceSummary{Blocked: 1},
					Attention: []controlplane.HostAttentionSummary{{
						RepositoryID:   "repo-remote",
						RepositoryName: "sergii/specview",
						Signals:        []string{"1 failed Evidence record"},
					}},
				},
			},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{
				{
					GroupID: "group:specview",
					Name:    "sergii/specview",
					Active:  true,
					Agents:  []string{"Codex"},
					Instances: []federation.SourcedInstance{
						{HostID: "host:550e8400-e29b-41d4-a716-446655440000", Hostname: "laptop", ObservedAt: now, RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:local", SourceRepositoryID: "repo-local", Name: "sergii/specview", Root: "/work/specview", Active: true}},
						{HostID: remoteHostID, Hostname: "devbox", ObservedAt: now.Add(-time.Minute), RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:remote", SourceRepositoryID: "repo-remote", Name: "sergii/specview", Root: "/srv/specview", Active: false}},
					},
				},
				{
					GroupID: "group:other",
					Name:    "sergii/other",
					Instances: []federation.SourcedInstance{{
						HostID: "host:550e8400-e29b-41d4-a716-446655440000", Hostname: "laptop", ObservedAt: now, RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:other", SourceRepositoryID: "repo-other", Name: "sergii/other", Root: "/work/other"},
					}},
				},
			},
		},
	}
	reader := &stubFederationReader{projection: projection}
	server := NewWithFederation(stubReader{}, reader, "test")

	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_host","arguments":{"host_id":"` + remoteHostID + `"}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("unexpected federation Host response: %#v", responses)
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if call.IsError {
		t.Fatalf("federation Host tool failed: %#v", call)
	}

	expected, err := projectFederationHost(projection, remoteHostID)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, expected))
	assertJSONEquivalent(t, []byte(call.Content[0].Text), mustJSON(t, expected))
	if reader.calls != 1 {
		t.Fatalf("federation Build calls = %d, want 1", reader.calls)
	}
	if !strings.Contains(call.Content[0].Text, `"freshness": "unreachable"`) || !strings.Contains(call.Content[0].Text, `"control_plane"`) {
		t.Fatalf("source-attributed Host facts missing: %s", call.Content[0].Text)
	}
	if strings.Contains(call.Content[0].Text, `"instance:local"`) || strings.Contains(call.Content[0].Text, `"instance:other"`) {
		t.Fatalf("federation Host tool leaked another Host repository instance: %s", call.Content[0].Text)
	}
}

func TestFederationHostToolRejectsInvalidArgumentsBeforeReading(t *testing.T) {
	for name, arguments := range map[string]string{
		"missing": `{}`,
		"blank":   `{"host_id":"   "}`,
		"extra":   `{"host_id":"host:fixture","extra":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			reader := &stubFederationReader{}
			server := NewWithFederation(stubReader{}, reader, "test")
			var output bytes.Buffer
			input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_host","arguments":` + arguments + `}}` + "\n"
			if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
				t.Fatal(err)
			}
			responses := decodeResponses(t, output.String())
			if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != invalidParamsCode {
				t.Fatalf("unexpected strict host_id response: %#v", responses)
			}
			if reader.calls != 0 {
				t.Fatalf("federation reader called for invalid arguments: %d", reader.calls)
			}
		})
	}
}

func TestFederationHostToolUnknownHostIsReadOnlyToolError(t *testing.T) {
	reader := &stubFederationReader{projection: federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		Hosts: []federationruntime.HostStatus{{
			Source: federationruntime.HostSourceLocal,
			HostID: "host:known",
		}},
	}}
	server := NewWithFederation(stubReader{}, reader, "test")
	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_host","arguments":{"host_id":"host:known-suffix"}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("unknown Host must remain a tool result: %#v", responses)
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if !call.IsError || !strings.Contains(call.Content[0].Text, `host:known-suffix`) || !strings.Contains(call.Content[0].Text, "not found") {
		t.Fatalf("unexpected unknown Host tool result: %#v", call)
	}
	if reader.calls != 1 {
		t.Fatalf("federation Build calls = %d, want 1", reader.calls)
	}
}
