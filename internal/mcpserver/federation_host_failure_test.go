package mcpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFederationHostToolProjectionFailureRemainsToolError(t *testing.T) {
	reader := &stubFederationReader{err: context.DeadlineExceeded}
	server := NewWithFederation(stubReader{}, reader, "test")
	var output bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_federation_host","arguments":{"host_id":"host:fixture"}}}` + "\n"
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	responses := decodeResponses(t, output.String())
	if len(responses) != 1 || responses[0].Error != nil {
		t.Fatalf("projection failure must remain a tool result: %#v", responses)
	}
	var call decodedToolResult
	decodeResult(t, responses[0], &call)
	if !call.IsError || len(call.Content) != 1 || !strings.Contains(call.Content[0].Text, "deadline") {
		t.Fatalf("unexpected projection failure tool result: %#v", call)
	}
	if reader.calls != 1 {
		t.Fatalf("federation Build calls = %d, want 1", reader.calls)
	}
}
