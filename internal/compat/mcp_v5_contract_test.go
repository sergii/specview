package compat_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMCPV5ToolContractIsStrictlyAdditive(t *testing.T) {
	var v4 mcpToolContractFixture
	var v5 mcpToolContractFixture
	if err := json.Unmarshal(readFixture(t, "mcp/v4-tools.json"), &v4); err != nil {
		t.Fatalf("decode MCP v4 tools: %v", err)
	}
	if err := json.Unmarshal(readFixture(t, "mcp/v5-tools.json"), &v5); err != nil {
		t.Fatalf("decode MCP v5 tools: %v", err)
	}
	if v5.SchemaVersion != 5 || v5.ProtocolVersion != v4.ProtocolVersion {
		t.Fatalf("unexpected MCP v5 metadata: %#v", v5)
	}
	if len(v5.Tools) != len(v4.Tools)+1 {
		t.Fatalf("MCP v5 tools = %d, want v4 + 1 (%d)", len(v5.Tools), len(v4.Tools)+1)
	}

	v5ByName := make(map[string][]string, len(v5.Tools))
	for _, tool := range v5.Tools {
		if _, exists := v5ByName[tool.Name]; exists {
			t.Fatalf("duplicate MCP v5 tool %q", tool.Name)
		}
		v5ByName[tool.Name] = tool.Arguments
	}
	for _, tool := range v4.Tools {
		arguments, ok := v5ByName[tool.Name]
		if !ok || !reflect.DeepEqual(arguments, tool.Arguments) {
			t.Fatalf("existing MCP tool changed: v4=%#v v5 arguments=%#v", tool, arguments)
		}
	}
	arguments, ok := v5ByName["get_federation_host"]
	if !ok || len(arguments) != 1 || arguments[0] != "host_id" {
		t.Fatalf("unexpected additive federation Host arguments: %#v", arguments)
	}
}
