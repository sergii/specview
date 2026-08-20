package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/specs"
)

func TestSpecificationCardNavigation(t *testing.T) {
	root := t.TempDir()
	specRoot := filepath.Join(root, "specs")
	if err := os.MkdirAll(specRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	const filename = "H07-git-status-observation.md"
	const document = `---
specview:
  status: new
---

# Git status observation

Observe useful Git state.
`
	if err := os.WriteFile(filepath.Join(specRoot, filename), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}

	store := specs.NewStore(specRoot, "*.md")
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Version: 1,
		Project: config.Project{Name: "Specview", Root: "."},
		Specs:   config.Specs{Path: "specs", Pattern: "*.md"},
		Server:  config.Server{Host: "127.0.0.1", Port: 7331},
	}
	server := NewServer(root, cfg, store, NewHub())

	board := httptest.NewRecorder()
	server.index(board, httptest.NewRequest(http.MethodGet, "/", nil))
	if board.Code != http.StatusOK {
		t.Fatalf("board status = %d, want %d", board.Code, http.StatusOK)
	}
	wantLink := `/spec?path=H07-git-status-observation.md`
	if !strings.Contains(board.Body.String(), wantLink) {
		t.Fatalf("board does not contain detail link %q", wantLink)
	}
	if !strings.Contains(board.Body.String(), `class="dense-id">H07</span>`) {
		t.Fatal("board does not render explicit stable display ID H07")
	}

	detail := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, wantLink, nil)
	server.detail(detail, request)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detail.Code, http.StatusOK)
	}
	if !strings.Contains(detail.Body.String(), "Git status observation") {
		t.Fatal("detail page does not contain specification title")
	}
}

func TestSpecDisplayID(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"H07-git-status-observation.md", "H07"},
		{"AUTH-03-login-flow.md", "AUTH-03"},
		{"api12-webhooks.md", "API12"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := specDisplayID(tt.path); got != tt.want {
				t.Fatalf("specDisplayID(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestSpecDisplayIDFallbackIsStable(t *testing.T) {
	path := "payments/retry-semantics.md"
	first := specDisplayID(path)
	second := specDisplayID(path)
	if first != second {
		t.Fatalf("fallback ID changed between calls: %q != %q", first, second)
	}
	if len(first) != 5 {
		t.Fatalf("fallback ID length = %d, want 5", len(first))
	}
	if first == specDisplayID("payments/another-specification.md") {
		t.Fatal("different paths unexpectedly received the same fallback ID")
	}
}
