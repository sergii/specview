package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	projectKey := server.projects[0].Key

	board := httptest.NewRecorder()
	server.index(board, httptest.NewRequest(http.MethodGet, "/", nil))
	if board.Code != http.StatusOK {
		t.Fatalf("board status = %d, want %d", board.Code, http.StatusOK)
	}
	body := board.Body.String()
	if !strings.Contains(body, "project="+projectKey) || !strings.Contains(body, "path=H07-git-status-observation.md") {
		t.Fatal("board does not contain project-scoped specification link")
	}
	if !strings.Contains(body, `class="dense-id">H07</span>`) {
		t.Fatal("board does not render explicit stable display ID H07")
	}

	wantLink := "/spec?project=" + projectKey + "&path=" + filename
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

func TestCompactProjectPathKeepsParentAndCurrentDirectory(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "Users", "serhii", "repos", "sergii", "specview")
	if got := compactProjectPath(root); got != "sergii/specview" {
		t.Fatalf("compactProjectPath(%q) = %q, want sergii/specview", root, got)
	}
}

func TestGraphProjectionResolvesSpecificationRelations(t *testing.T) {
	root := t.TempDir()
	specRoot := filepath.Join(root, "specs")
	if err := os.MkdirAll(specRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specRoot, "H01-base.md"), []byte("# Base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specRoot, "H02-dependent.md"), []byte("---\nspecview:\n  status: in_progress\n  depends_on: [H01]\n---\n# Dependent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := specs.NewStore(specRoot, "*.md")
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Version: 1,
		Project: config.Project{Name: "Graph", Root: "."},
		Specs:   config.Specs{Path: "specs", Pattern: "*.md"},
		Server:  config.Server{Host: "127.0.0.1", Port: 7331},
	}
	server := NewServer(root, cfg, store, NewHub())
	graph := server.graphData(time.Now())
	if len(graph.Nodes) != 2 {
		t.Fatalf("graph nodes = %d, want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("graph edges = %d, want 1", len(graph.Edges))
	}
	edge := graph.Edges[0]
	if edge.Missing || edge.Type != "depends_on" {
		t.Fatalf("unexpected graph edge: %#v", edge)
	}
	if !strings.Contains(edge.From, "H01-base.md") || !strings.Contains(edge.To, "H02-dependent.md") {
		t.Fatalf("unexpected graph direction: %#v", edge)
	}
}

func TestGraphPageRendersBothModes(t *testing.T) {
	root := t.TempDir()
	specRoot := filepath.Join(root, "specs")
	if err := os.MkdirAll(specRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	store := specs.NewStore(specRoot, "*.md")
	if err := store.Refresh(); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Version: 1,
		Project: config.Project{Name: "Graph", Root: "."},
		Specs:   config.Specs{Path: "specs", Pattern: "*.md"},
		Server:  config.Server{Host: "127.0.0.1", Port: 7331},
	}
	server := NewServer(root, cfg, store, NewHub())

	for _, tt := range []struct {
		url, want string
	}{
		{"/graph", `data-mode="2d"`},
		{"/graph?mode=3d", `data-mode="3d"`},
	} {
		response := httptest.NewRecorder()
		server.graphPage(response, httptest.NewRequest(http.MethodGet, tt.url, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d", tt.url, response.Code)
		}
		if !strings.Contains(response.Body.String(), tt.want) {
			t.Fatalf("%s does not render %s", tt.url, tt.want)
		}
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
