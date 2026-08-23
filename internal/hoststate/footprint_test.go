package hoststate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatFootprintBytes(t *testing.T) {
	tests := map[int64]string{
		300 * 1024:                   "300 KB",
		2 * 1024 * 1024:              "2 MB",
		812 * 1024 * 1024:            "812 MB",
		15 * 1024 * 1024 * 1024 / 10: "1.5 GB",
	}
	for input, want := range tests {
		if got := formatFootprintBytes(input); got != want {
			t.Fatalf("formatFootprintBytes(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestMeasureRepositoryFootprintSeparatesWorkingTreeAndGit(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}

	payload := make([]byte, 4096)
	if err := os.WriteFile(filepath.Join(root, "payload.bin"), payload, 0o644); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	measuredAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	footprint, err := measureRepositoryFootprint(root, measuredAt)
	if err != nil {
		t.Fatalf("measureRepositoryFootprint: %v", err)
	}
	if footprint.WorkingTreeBytes != int64(len(payload)) {
		t.Fatalf("working tree bytes = %d, want %d", footprint.WorkingTreeBytes, len(payload))
	}
	if footprint.GitBytes <= 0 {
		t.Fatalf("git bytes = %d, want > 0", footprint.GitBytes)
	}
	if footprint.TotalBytes != footprint.WorkingTreeBytes+footprint.GitBytes {
		t.Fatalf("total bytes = %d, components = %d + %d", footprint.TotalBytes, footprint.WorkingTreeBytes, footprint.GitBytes)
	}
	if !footprint.MeasuredAt.Equal(measuredAt) {
		t.Fatalf("measured at = %v, want %v", footprint.MeasuredAt, measuredAt)
	}
}

func TestRepositoryFootprintCacheKeepsRecentMeasurement(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, "first.bin"), make([]byte, 1024), 0o644); err != nil {
		t.Fatalf("write first payload: %v", err)
	}

	cache := repositoryFootprintCache{entries: make(map[string]footprintCacheEntry)}
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	first := cache.measure(root, now)
	if !first.Available() {
		t.Fatal("first footprint unavailable")
	}

	if err := os.WriteFile(filepath.Join(root, "second.bin"), make([]byte, 4096), 0o644); err != nil {
		t.Fatalf("write second payload: %v", err)
	}
	second := cache.measure(root, now.Add(time.Minute))
	if second.TotalBytes != first.TotalBytes {
		t.Fatalf("cached total = %d, want %d", second.TotalBytes, first.TotalBytes)
	}
}
