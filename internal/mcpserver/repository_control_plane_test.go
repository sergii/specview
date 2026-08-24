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

type repositoryControlPlaneStubReader struct {
	stubReader
	controlPlane controlplane.GetRepositoryControlPlaneResult
}

func (s repositoryControlPlaneStubReader) GetRepositoryControlPlane(context.Context, string) (controlplane.GetRepositoryControlPlaneResult, error) {
	return s.controlPlane, s.err
}

func TestGetRepositoryControlPlaneToolReturnsStructuredProjection(t *testing.T) {
	now := time.Date(2026, 8, 24, 16, 0, 0, 0, time.UTC)
	projection := controlplane.GetRepositoryControlPlaneResult{
		SchemaVersion:  1,
		Host:           "laptop.local",
		RepositoryID:   "repo-a",
		RepositoryName: "sergii/specview",
		Intent: controlplane.RepositoryIntentSummary{
			Total:      3,
			InProgress: 1,
			Done:       2,
		},
		Execution: controlplane.RepositoryExecutionSummary{
			Active: 1,
			Latest: &executionhistory.Entry{
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
		Evidence: controlplane.RepositoryEvidenceOverviewSummary{
			Total:  4,
			Passed: 4,
		},
		Acceptance: controlplane.RepositoryAcceptanceOverviewSummary{
			Configured:    true,
			EvidenceCount: 4,
			Accepted:      2,
			Waiting:       1,
		},
	}
	server := New(repositoryControlPlaneStubReader{controlPlane: projection}, "test")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"tools","method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_repository_control_plane","arguments":{"repository_id":"repo-a"}}}`,
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
		if tool.Name == "get_repository_control_plane" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("repository control-plane tool missing: %#v", list.Tools)
	}

	var call decodedToolResult
	decodeResult(t, responses[1], &call)
	if call.IsError {
		t.Fatalf("control-plane tool returned error: %#v", call)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, projection))
}
