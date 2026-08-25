package compat_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMCPV6ToolContractIsStrictlyAdditive(t *testing.T) {
	var v5 mcpToolContractFixture
	var v6 mcpToolContractFixture
	if err := json.Unmarshal(readFixture(t, "mcp/v5-tools.json"), &v5); err != nil {
		t.Fatalf("decode MCP v5 tools: %v", err)
	}
	if err := json.Unmarshal(readFixture(t, "mcp/v6-tools.json"), &v6); err != nil {
		t.Fatalf("decode MCP v6 tools: %v", err)
	}
	if v6.SchemaVersion != 6 || v6.ProtocolVersion != v5.ProtocolVersion {
		t.Fatalf("unexpected MCP v6 metadata: %#v", v6)
	}
	if len(v6.Tools) != len(v5.Tools)+1 {
		t.Fatalf("MCP v6 tools = %d, want v5 + 1 (%d)", len(v6.Tools), len(v5.Tools)+1)
	}

	v6ByName := make(map[string][]string, len(v6.Tools))
	for _, tool := range v6.Tools {
		if _, exists := v6ByName[tool.Name]; exists {
			t.Fatalf("duplicate MCP v6 tool %q", tool.Name)
		}
		v6ByName[tool.Name] = tool.Arguments
	}
	for _, tool := range v5.Tools {
		arguments, ok := v6ByName[tool.Name]
		if !ok || !reflect.DeepEqual(arguments, tool.Arguments) {
			t.Fatalf("existing MCP tool changed: v5=%#v v6 arguments=%#v", tool, arguments)
		}
	}
	arguments, ok := v6ByName["get_federation_repository"]
	if !ok || !reflect.DeepEqual(arguments, []string{"host_id", "instance_id"}) {
		t.Fatalf("unexpected additive federation repository arguments: %#v", arguments)
	}
}
