package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
)

type stubFederationReader struct {
	projection federationruntime.Projection
	err        error
	calls      int
}

func (s *stubFederationReader) Build(context.Context) (federationruntime.Projection, error) {
	s.calls++
	return s.projection, s.err
}

func TestFederationStatusToolReturnsSharedProjectionContract(t *testing.T) {
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{
			{
				Source:      federationruntime.HostSourceLocal,
				HostID:      "host:550e8400-e29b-41d4-a716-446655440000",
				Hostname:    "laptop",
				HasSnapshot: true,
				ControlPlane: &controlplane.GetHostControlPlaneResult{
					SchemaVersion: controlplane.SchemaVersion,
					Host:          "laptop",
					Execution:     controlplane.HostExecutionSummary{ActiveSessions: 1},
					Attention:     []controlplane.HostAttentionSummary{},
				},
			},
			{
				Source:      federationruntime.HostSourcePeer,
				Peer:        "devbox",
				HostID:      "host:550e8400-e29b-41d4-a716-446655440001",
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
					Attention:     []controlplane.HostAttentionSummary{},
				},
			},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:fixture",
				Name:    "sergii/specview",
				Active:  true,
			}},
		},
	}
	reader := &stubFederationReader{projection: projection}
	server := NewWithFederation(stubReader{}, reader, "test")

	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_status","arguments":{}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if call.IsError || len(call.Content) != 1 {
		t.Fatalf("unexpected federation tool result: %#v", call)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, projection))
	assertJSONEquivalent(t, []byte(call.Content[0].Text), mustJSON(t, projection))
	if !strings.Contains(call.Content[0].Text, `"control_plane"`) {
		t.Fatalf("federation MCP result lost per-Host control plane: %s", call.Content[0].Text)
	}
	if reader.calls != 1 {
		t.Fatalf("federation Build calls = %d, want 1", reader.calls)
	}
}

func TestFederationStatusToolRejectsArgumentsBeforeReading(t *testing.T) {
	reader := &stubFederationReader{}
	server := NewWithFederation(stubReader{}, reader, "test")
	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_status","arguments":{"extra":true}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != invalidParamsCode {
		t.Fatalf("unexpected strict-arguments response: %#v", responses)
	}
	if reader.calls != 0 {
		t.Fatalf("federation reader called for invalid arguments: %d", reader.calls)
	}
}

func TestFederationStatusFailureIsAReadOnlyToolError(t *testing.T) {
	reader := &stubFederationReader{err: errors.New("projection unavailable")}
	server := NewWithFederation(stubReader{}, reader, "test")
	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_status","arguments":{}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("domain failure must remain a tool result: %#v", responses)
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if !call.IsError || len(call.Content) != 1 || !strings.Contains(call.Content[0].Text, "projection unavailable") {
		t.Fatalf("unexpected federation domain error: %#v", call)
	}
}
