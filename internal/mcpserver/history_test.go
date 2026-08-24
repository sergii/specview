package mcpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/executionhistory"
)

type historyStubReader struct {
	stubReader
	history executionhistory.Projection
}

func (s historyStubReader) GetExecutionHistory(context.Context) (executionhistory.Projection, error) {
	return s.history, s.err
}

func TestGetExecutionHistoryToolReturnsStructuredProjection(t *testing.T) {
	now := time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC)
	projection := executionhistory.Projection{
		SchemaVersion: executionhistory.SchemaVersion,
		Hostname:      "laptop.local",
		Entries: []executionhistory.Entry{{
			RepositoryID:   "repo-a",
			RepositoryName: "sergii/specview",
			RepositoryRoot: "/work/sergii/specview",
			SessionID:      "session-a",
			IdentityKind:   "logical",
			Adapter:        "codex",
			Agent:          "Codex",
			ProcessIDs:     []int{10, 11},
			StartedAt:      now.Add(-time.Minute),
			LastSeenAt:     now,
			Active:         true,
		}},
	}
	server := New(historyStubReader{history: projection}, "test")
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_execution_history","arguments":{}}}` + "\n"
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if call.IsError {
		t.Fatalf("history tool returned error: %#v", call)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, projection))
}
