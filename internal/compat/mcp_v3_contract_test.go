package compat_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMCPV3ToolContractIsStrictlyAdditive(t *testing.T) {
	var v2 mcpToolContractFixture
	var v3 mcpToolContractFixture
	if err := json.Unmarshal(readFixture(t, "mcp/v2-tools.json"), &v2); err != nil {
		t.Fatalf("decode MCP v2 tools: %v", err)
	}
	if err := json.Unmarshal(readFixture(t, "mcp/v3-tools.json"), &v3); err != nil {
		t.Fatalf("decode MCP v3 tools: %v", err)
	}
	if v3.SchemaVersion != 3 || v3.ProtocolVersion != v2.ProtocolVersion {
		t.Fatalf("unexpected MCP v3 metadata: %#v", v3)
	}
	if len(v3.Tools) != len(v2.Tools)+1 {
		t.Fatalf("MCP v3 tools = %d, want v2 + 1 (%d)", len(v3.Tools), len(v2.Tools)+1)
	}

	v3ByName := make(map[string][]string, len(v3.Tools))
	for _, tool := range v3.Tools {
		if _, exists := v3ByName[tool.Name]; exists {
			t.Fatalf("duplicate MCP v3 tool %q", tool.Name)
		}
		v3ByName[tool.Name] = tool.Arguments
	}
	for _, tool := range v2.Tools {
		arguments, ok := v3ByName[tool.Name]
		if !ok || !reflect.DeepEqual(arguments, tool.Arguments) {
			t.Fatalf("existing MCP tool changed: v2=%#v v3 arguments=%#v", tool, arguments)
		}
	}
	arguments, ok := v3ByName["get_execution_history"]
	if !ok || len(arguments) != 0 {
		t.Fatalf("unexpected additive history tool arguments: %#v", arguments)
	}
}
