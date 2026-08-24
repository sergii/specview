package compat_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

type mcpToolContractFixture struct {
	SchemaVersion   int    `json:"schema_version"`
	ProtocolVersion string `json:"protocol_version"`
	Tools           []struct {
		Name      string   `json:"name"`
		Arguments []string `json:"arguments"`
	} `json:"tools"`
}

func TestMCPV2ToolContractIsStrictlyAdditive(t *testing.T) {
	var v1 mcpToolContractFixture
	var v2 mcpToolContractFixture
	if err := json.Unmarshal(readFixture(t, "mcp/v1-tools.json"), &v1); err != nil {
		t.Fatalf("decode MCP v1 tools: %v", err)
	}
	if err := json.Unmarshal(readFixture(t, "mcp/v2-tools.json"), &v2); err != nil {
		t.Fatalf("decode MCP v2 tools: %v", err)
	}
	if v2.SchemaVersion != 2 || v2.ProtocolVersion != v1.ProtocolVersion {
		t.Fatalf("unexpected MCP v2 metadata: %#v", v2)
	}
	if len(v2.Tools) != len(v1.Tools)+1 {
		t.Fatalf("MCP v2 tools = %d, want v1 + 1 (%d)", len(v2.Tools), len(v1.Tools)+1)
	}
	for i := range v1.Tools {
		if v2.Tools[i].Name != v1.Tools[i].Name || !reflect.DeepEqual(v2.Tools[i].Arguments, v1.Tools[i].Arguments) {
			t.Fatalf("existing MCP tool %d changed: v1=%#v v2=%#v", i, v1.Tools[i], v2.Tools[i])
		}
	}
	added := v2.Tools[len(v2.Tools)-1]
	if added.Name != "get_federation_status" || len(added.Arguments) != 0 {
		t.Fatalf("unexpected additive MCP tool: %#v", added)
	}
}
