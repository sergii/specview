package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectConventionWithoutWriting(t *testing.T) {
	tests := []struct {
		name    string
		marker  string
		adapter string
		label   string
	}{
		{name: "spec kit", marker: ".specify", adapter: AdapterGitHubSpecKit, label: "GitHub Spec Kit"},
		{name: "openspec", marker: filepath.Join("openspec", "changes"), adapter: AdapterOpenSpec, label: "OpenSpec"},
		{name: "kiro", marker: filepath.Join(".kiro", "specs"), adapter: AdapterKiro, label: "Kiro"},
		{name: "bmad", marker: "_bmad-output", adapter: AdapterBMAD, label: "BMAD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, tt.marker), 0o755); err != nil {
				t.Fatal(err)
			}
			got, err := DetectConvention(root)
			if err != nil {
				t.Fatal(err)
			}
			if got.Adapter != tt.adapter || got.Label != tt.label || !got.Recognized {
				t.Fatalf("unexpected convention: %#v", got)
			}
			if tt.adapter == AdapterKiro || tt.adapter == AdapterBMAD {
				if got.Supported {
					t.Fatalf("%s parser should be detection-only in this slice", tt.adapter)
				}
			}
		})
	}
}

func TestPlainSpecsDirectoryIsAmbiguous(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := DetectConvention(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Recognized {
		t.Fatalf("plain specs directory must not select an adapter: %#v", got)
	}
}
