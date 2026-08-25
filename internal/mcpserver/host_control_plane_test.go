package mcpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/executionhistory"
)

type hostControlPlaneStubReader struct {
	stubReader
	controlPlane controlplane.GetHostControlPlaneResult
}

func (s hostControlPlaneStubReader) GetHostControlPlane(context.Context) (controlplane.GetHostControlPlaneResult, error) {
	return s.controlPlane, s.err
}

func TestGetHostControlPlaneToolReturnsStructuredProjection(t *testing.T) {
	now := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	projection := controlplane.GetHostControlPlaneResult{
		SchemaVersion: 1,
		Host:          "laptop.local",
		Intent: controlplane.HostIntentSummary{
			ManagedRepositories: 2,
			WorkItems:           4,
			InProgress:          1,
			Done:                3,
		},
		Execution: controlplane.HostExecutionSummary{
			ActiveSessions:     1,
			ActiveRepositories: 1,
			HasLatest:          true,
			Latest: executionhistory.Entry{
				RepositoryID:   "repo-a",
				RepositoryName: "sergii/specview",
				RepositoryRoot: "/work/sergii/specview",
				SessionID:      "session-a",
				IdentityKind:   "logical",
				Adapter:        "codex",
				Agent:          "Codex",
				StartedAt:      now.Add(-time.Minute),
				LastSeenAt:     now,
				Active:         true,
			},
		},
		Evidence: controlplane.HostEvidenceSummary{
			Total:  5,
			Passed: 4,
			Failed: 1,
		},
		Acceptance: controlplane.HostAcceptanceSummary{
			ConfiguredRepositories: 2,
			Accepted:               3,
			Blocked:                1,
		},
		Attention: []controlplane.HostAttentionSummary{{
			RepositoryID:   "repo-a",
			RepositoryName: "sergii/specview",
			LastSeenAt:     now,
			Signals:        []string{"1 failed Evidence record", "1 blocked Acceptance item"},
		}},
	}
	server := New(hostControlPlaneStubReader{controlPlane: projection}, "test")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"tools","method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_host_control_plane","arguments":{}}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 2 {
		t.Fatalf("responses = %d, want 2", len(responses))
	}

	var list struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	decodeResult(t, responses[0], &list)
	found := false
	for _, tool := range list.Tools {
		if tool.Name == "get_host_control_plane" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Host control-plane tool missing: %#v", list.Tools)
	}

	var call decodedToolResult
	decodeResult(t, responses[1], &call)
	if call.IsError {
		t.Fatalf("Host control-plane tool returned error: %#v", call)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, projection))
}
