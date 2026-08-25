package mcpserver

import "testing"

func TestFederationDrillDownToolsAreAdvertisedOnlyWithFederationReader(t *testing.T) {
	configured := toolDefinitionsForServer(stubReader{}, true)
	if !hasToolDefinition(configured, "get_federation_host") || !hasToolDefinition(configured, "get_federation_repository") {
		t.Fatalf("configured federation discovery lost drill-down tools: %#v", configured)
	}

	legacy := toolDefinitionsForServer(stubReader{}, false)
	if hasToolDefinition(legacy, "get_federation_host") || hasToolDefinition(legacy, "get_federation_repository") {
		t.Fatalf("legacy discovery exposed federation drill-down tools without a reader: %#v", legacy)
	}
}

func hasToolDefinition(definitions []map[string]any, want string) bool {
	for _, definition := range definitions {
		if name, _ := definition["name"].(string); name == want {
			return true
		}
	}
	return false
}
