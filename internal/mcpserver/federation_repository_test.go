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
	"github.com/sergii/specview/internal/identity"
)

func TestFederationRepositoryToolReturnsExactHostScopedInstance(t *testing.T) {
	now := time.Date(2026, 8, 25, 8, 30, 0, 0, time.UTC)
	remoteHostID := "host:550e8400-e29b-41d4-a716-446655440001"
	selected := federation.SourcedInstance{
		HostID:     remoteHostID,
		Hostname:   "devbox",
		ObservedAt: now.Add(-time.Minute),
		RepositoryInstance: federation.RepositoryInstance{
			InstanceID:         "instance:remote",
			SourceRepositoryID: "repo-remote",
			Name:               "sergii/specview",
			Root:               "/srv/specview",
			Fingerprint: identity.RepositoryFingerprint{
				ExplicitID:      "specview-project",
				GitRemote:       "git@github.com:sergii/specview.git",
				ForgeProvider:   "github",
				ForgeRepository: "sergii/specview",
			},
			Active: true,
			Agents: []string{"Codex"},
			Sessions: []federation.Session{{
				ID:           "session-remote",
				Adapter:      "codex",
				Agent:        "Codex",
				WorktreeRoot: "/srv/specview",
				CWD:          "/srv/specview/specs",
				StartedAt:    "2026-08-25T08:00:00Z",
			}},
			Worktrees: []federation.Worktree{{
				Path:       "/srv/specview",
				Branch:     "main",
				Head:       "abc123",
				DirtyCount: 1,
				Upstream:   "origin/main",
				Ahead:      2,
				Behind:     1,
				LastCommit: "abc123",
			}},
			Warnings: []string{"fixture warning"},
		},
	}
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		GeneratedAt:   now,
		Hosts: []federationruntime.HostStatus{
			{Source: federationruntime.HostSourceLocal, HostID: "host:550e8400-e29b-41d4-a716-446655440000", Hostname: "laptop", HasSnapshot: true},
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
					Execution:     controlplane.HostExecutionSummary{ActiveSessions: 1},
					Attention:     []controlplane.HostAttentionSummary{},
				},
			},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			GeneratedAt:   now,
			Repositories: []federation.RepositoryGroup{
				{
					GroupID:   "group:specview",
					Name:      "sergii/specview",
					Active:    true,
					Agents:    []string{"Codex", "Claude"},
					Instances: []federation.SourcedInstance{{HostID: "host:550e8400-e29b-41d4-a716-446655440000", RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:local", Name: "sergii/specview"}}, selected},
				},
				{
					GroupID: "group:other",
					Name:    "sergii/other",
					Instances: []federation.SourcedInstance{{
						HostID: remoteHostID, RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:other", Name: "sergii/other"},
					}},
				},
			},
		},
	}
	reader := &stubFederationReader{projection: projection}
	server := NewWithFederation(stubReader{}, reader, "test")

	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_repository","arguments":{"host_id":"` + remoteHostID + `","instance_id":"instance:remote"}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("unexpected federation repository response: %#v", responses)
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if call.IsError {
		t.Fatalf("federation repository tool failed: %#v", call)
	}
	expected, err := projectFederationRepository(projection, remoteHostID, "instance:remote")
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, expected))
	assertJSONEquivalent(t, []byte(call.Content[0].Text), mustJSON(t, expected))
	if reader.calls != 1 {
		t.Fatalf("federation Build calls = %d, want 1", reader.calls)
	}
	for _, required := range []string{`"freshness": "unreachable"`, `"session-remote"`, `"/srv/specview"`, `"specview-project"`, `"group:specview"`} {
		if !strings.Contains(call.Content[0].Text, required) {
			t.Fatalf("federation repository fact %s missing: %s", required, call.Content[0].Text)
		}
	}
	if strings.Contains(call.Content[0].Text, `"instance:local"`) || strings.Contains(call.Content[0].Text, `"instance:other"`) {
		t.Fatalf("repository tool leaked another instance: %s", call.Content[0].Text)
	}
}

func TestFederationRepositoryToolRejectsInvalidArgumentsBeforeReading(t *testing.T) {
	for name, arguments := range map[string]string{
		"missing":          `{}`,
		"blank-host":       `{"host_id":"   ","instance_id":"instance:fixture"}`,
		"missing-instance": `{"host_id":"host:fixture"}`,
		"blank-instance":   `{"host_id":"host:fixture","instance_id":"   "}`,
		"extra":            `{"host_id":"host:fixture","instance_id":"instance:fixture","extra":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			reader := &stubFederationReader{}
			server := NewWithFederation(stubReader{}, reader, "test")
			var output bytes.Buffer
			input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_repository","arguments":` + arguments + `}}` + "\n"
			if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
				t.Fatal(err)
			}
			responses := decodeResponses(t, output.String())
			if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != invalidParamsCode {
				t.Fatalf("unexpected strict federation repository response: %#v", responses)
			}
			if reader.calls != 0 {
				t.Fatalf("federation reader called for invalid arguments: %d", reader.calls)
			}
		})
	}
}

func TestFederationRepositoryToolRejectsUnknownOrWrongHostSelection(t *testing.T) {
	projection := federationruntime.Projection{
		SchemaVersion: federationruntime.ProjectionSchemaVersion,
		Hosts: []federationruntime.HostStatus{
			{Source: federationruntime.HostSourceLocal, HostID: "host:a"},
			{Source: federationruntime.HostSourcePeer, HostID: "host:b"},
		},
		Federation: federation.Projection{
			SchemaVersion: federation.ProjectionSchemaVersion,
			Repositories: []federation.RepositoryGroup{{
				GroupID: "group:fixture",
				Name:    "fixture/repo",
				Instances: []federation.SourcedInstance{{
					HostID: "host:b", RepositoryInstance: federation.RepositoryInstance{InstanceID: "instance:b"},
				}},
			}},
		},
	}
	cases := map[string][2]string{
		"unknown-host":     {"host:missing", "instance:b"},
		"unknown-instance": {"host:b", "instance:missing"},
		"wrong-host":       {"host:a", "instance:b"},
	}
	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			reader := &stubFederationReader{projection: projection}
			server := NewWithFederation(stubReader{}, reader, "test")
			var output bytes.Buffer
			input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_repository","arguments":{"host_id":"` + pair[0] + `","instance_id":"` + pair[1] + `"}}}` + "\n"
			if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
				t.Fatal(err)
			}
			responses := decodeResponses(t, output.String())
			if len(responses) != 1 || responses[0].Error != nil {
				t.Fatalf("unknown selection must remain a tool result: %#v", responses)
			}
			var call decodedToolResult
			decodeResult(t, responses[0], &call)
			if !call.IsError || !strings.Contains(call.Content[0].Text, "not found") {
				t.Fatalf("unexpected unknown selection result: %#v", call)
			}
			if reader.calls != 1 {
				t.Fatalf("federation Build calls = %d, want 1", reader.calls)
			}
		})
	}
}

func TestFederationRepositoryToolProjectionFailureIsReadOnlyToolError(t *testing.T) {
	reader := &stubFederationReader{err: errors.New("projection unavailable")}
	server := NewWithFederation(stubReader{}, reader, "test")
	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_repository","arguments":{"host_id":"host:fixture","instance_id":"instance:fixture"}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("projection failure must remain a tool result: %#v", responses)
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if !call.IsError || !strings.Contains(call.Content[0].Text, "projection unavailable") {
		t.Fatalf("unexpected projection failure result: %#v", call)
	}
	if reader.calls != 1 {
		t.Fatalf("federation Build calls = %d, want 1", reader.calls)
	}
}
