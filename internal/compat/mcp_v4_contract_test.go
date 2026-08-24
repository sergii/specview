package compat_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMCPV4ToolContractIsStrictlyAdditive(t *testing.T) {
	var v3 mcpToolContractFixture
	var v4 mcpToolContractFixture
	if err := json.Unmarshal(readFixture(t, "mcp/v3-tools.json"), &v3); err != nil {
		t.Fatalf("decode MCP v3 tools: %v", err)
	}
	if err := json.Unmarshal(readFixture(t, "mcp/v4-tools.json"), &v4); err != nil {
		t.Fatalf("decode MCP v4 tools: %v", err)
	}
	if v4.SchemaVersion != 4 || v4.ProtocolVersion != v3.ProtocolVersion {
		t.Fatalf("unexpected MCP v4 metadata: %#v", v4)
	}
	if len(v4.Tools) != len(v3.Tools)+1 {
		t.Fatalf("MCP v4 tools = %d, want v3 + 1 (%d)", len(v4.Tools), len(v3.Tools)+1)
	}

	v4ByName := make(map[string][]string, len(v4.Tools))
	for _, tool := range v4.Tools {
		if _, exists := v4ByName[tool.Name]; exists {
			t.Fatalf("duplicate MCP v4 tool %q", tool.Name)
		}
		v4ByName[tool.Name] = tool.Arguments
	}
	for _, tool := range v3.Tools {
		arguments, ok := v4ByName[tool.Name]
		if !ok || !reflect.DeepEqual(arguments, tool.Arguments) {
			t.Fatalf("existing MCP tool changed: v3=%#v v4 arguments=%#v", tool, arguments)
		}
	}
	arguments, ok := v4ByName["get_repository_control_plane"]
	if !ok || len(arguments) != 1 || arguments[0] != "repository_id" {
		t.Fatalf("unexpected additive repository control-plane arguments: %#v", arguments)
	}
}
