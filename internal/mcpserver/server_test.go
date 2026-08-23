package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sergii/specview/internal/controlplane"
)

type stubReader struct {
	repositories controlplane.ListRepositoriesResult
	repository   controlplane.GetRepositoryResult
	sessions     controlplane.ListActiveSessionsResult
	worktrees    controlplane.ListWorktreesResult
	err          error
}

func (s stubReader) ListRepositories(context.Context) (controlplane.ListRepositoriesResult, error) {
	return s.repositories, s.err
}

func (s stubReader) GetRepository(context.Context, string) (controlplane.GetRepositoryResult, error) {
	return s.repository, s.err
}

func (s stubReader) ListActiveSessions(context.Context) (controlplane.ListActiveSessionsResult, error) {
	return s.sessions, s.err
}

func (s stubReader) ListWorktrees(context.Context, string) (controlplane.ListWorktreesResult, error) {
	return s.worktrees, s.err
}

type decodedResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

func TestLegacyStdioInitializeToolsAndRepositoryContract(t *testing.T) {
	fixture := readRepositoryFixture(t)
	reader := stubReader{repositories: fixture}
	server := New(reader, "v0.0.2-test")

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"tools","method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_repositories","arguments":{}}}`,
	}, "\n") + "\n"

	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 3 {
		t.Fatalf("responses = %d, want 3; output=%s", len(responses), output.String())
	}

	var initialize map[string]any
	decodeResult(t, responses[0], &initialize)
	if initialize["protocolVersion"] != ProtocolVersion {
		t.Fatalf("protocolVersion = %#v", initialize["protocolVersion"])
	}
	serverInfo := initialize["serverInfo"].(map[string]any)
	if serverInfo["name"] != "specview" || serverInfo["version"] != "v0.0.2-test" {
		t.Fatalf("serverInfo = %#v", serverInfo)
	}

	var list struct {
		Tools []struct {
			Name        string         `json:"name"`
			Annotations map[string]any `json:"annotations"`
		} `json:"tools"`
	}
	decodeResult(t, responses[1], &list)
	if len(list.Tools) != 4 {
		t.Fatalf("tools = %#v", list.Tools)
	}
	for _, tool := range list.Tools {
		if tool.Annotations["readOnlyHint"] != true || tool.Annotations["destructiveHint"] != false {
			t.Fatalf("tool %s is not read-only: %#v", tool.Name, tool.Annotations)
		}
	}

	var call struct {
		Content []content       `json:"content"`
		Structured json.RawMessage `json:"structuredContent"`
		IsError bool            `json:"isError"`
	}
	decodeResult(t, responses[2], &call)
	if call.IsError || len(call.Content) != 1 || call.Content[0].Type != "text" {
		t.Fatalf("unexpected tool result: %#v", call)
	}
	assertJSONEquivalent(t, call.Structured, mustJSON(t, fixture))
	assertJSONEquivalent(t, []byte(call.Content[0].Text), mustJSON(t, fixture))
}

func TestModernDiscoveryCanFallBackToLegacyInitialize(t *testing.T) {
	server := New(stubReader{}, "test")
	input := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{}}` + "\n"
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != methodNotFoundCode {
		t.Fatalf("server/discover response = %#v", responses)
	}
}

func TestToolArgumentsAreStrictAndDomainFailuresStayToolErrors(t *testing.T) {
	server := New(stubReader{err: context.DeadlineExceeded}, "test")
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_repository","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_repositories","arguments":{"extra":true}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_active_sessions","arguments":{}}}`,
	}, "\n") + "\n"

	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 3 {
		t.Fatalf("responses = %#v", responses)
	}
	if responses[0].Error == nil || responses[0].Error.Code != invalidParamsCode {
		t.Fatalf("missing repository id must be protocol invalid params: %#v", responses[0])
	}
	if responses[1].Error == nil || responses[1].Error.Code != invalidParamsCode {
		t.Fatalf("unexpected arguments must be protocol invalid params: %#v", responses[1])
	}
	if responses[2].Error != nil {
		t.Fatalf("domain failure must not become JSON-RPC failure: %#v", responses[2].Error)
	}
	var tool toolResult
	decodeResult(t, responses[2], &tool)
	if !tool.IsError || len(tool.Content) != 1 || !strings.Contains(tool.Content[0].Text, "deadline") {
		t.Fatalf("unexpected domain tool error: %#v", tool)
	}
}

func TestParseErrorUsesNullID(t *testing.T) {
	server := New(stubReader{}, "test")
	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader("not-json\n"), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != parseErrorCode || string(responses[0].ID) != "null" {
		t.Fatalf("parse response = %#v", responses)
	}
}

func readRepositoryFixture(t *testing.T) controlplane.ListRepositoriesResult {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "contracts", "mcp", "v1-list-repositories.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result controlplane.ListRepositoriesResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func decodeResponses(t *testing.T, output string) []decodedResponse {
	t.Helper()
	var responses []decodedResponse
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response decodedResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("decode response %q: %v", line, err)
		}
		responses = append(responses, response)
	}
	return responses
}

func decodeResult(t *testing.T, response decodedResponse, destination any) {
	t.Helper()
	if response.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %#v", response.Error)
	}
	if err := json.Unmarshal(response.Result, destination); err != nil {
		t.Fatalf("decode result: %v; raw=%s", err, response.Result)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertJSONEquivalent(t *testing.T, left, right []byte) {
	t.Helper()
	var leftValue any
	var rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		t.Fatalf("decode left JSON: %v; %s", err, left)
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		t.Fatalf("decode right JSON: %v; %s", err, right)
	}
	leftCanonical, _ := json.Marshal(leftValue)
	rightCanonical, _ := json.Marshal(rightValue)
	if !bytes.Equal(leftCanonical, rightCanonical) {
		t.Fatalf("JSON mismatch\nleft:  %s\nright: %s", leftCanonical, rightCanonical)
	}
}
