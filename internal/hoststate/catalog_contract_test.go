package hoststate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCatalogV1GoldenFixtureMatchesWriterShape(t *testing.T) {
	path := catalogContractFixturePath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read catalog contract fixture: %v", err)
	}

	var persisted persistedCatalog
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("decode catalog contract fixture: %v", err)
	}
	if persisted.Version != catalogVersion {
		t.Fatalf("fixture version = %d, writer version = %d", persisted.Version, catalogVersion)
	}

	encoded, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		t.Fatalf("encode catalog contract fixture: %v", err)
	}
	encoded = append(encoded, '\n')
	if string(encoded) != string(data) {
		t.Fatalf("catalog v1 writer shape changed\n--- fixture ---\n%s\n--- encoded ---\n%s", data, encoded)
	}
}

func catalogContractFixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve catalog contract test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(root, "testdata", "contracts", "catalog", "v1.json")
}
