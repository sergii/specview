package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergii/specview/internal/activity"
	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/specs"
)

func TestBoardRendersFreshAgentActivity(t *testing.T) {
	root := t.TempDir()
	specRoot := filepath.Join(root, "specs")
	if err := os.MkdirAll(specRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specRoot, "H10-design.md"), []byte("---\nspecview:\n  status: in_progress\n---\n\n# Design\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	specStore := specs.NewStore(specRoot, "*.md")
	if err := specStore.Refresh(); err != nil {
		t.Fatal(err)
	}

	activityRoot := filepath.Join(root, activity.RelativeDir)
	if err := os.MkdirAll(activityRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	heartbeat := time.Now().UTC().Format(time.RFC3339Nano)
	record := fmt.Sprintf(`{
  "version": 1,
  "session_id": "session-1",
  "agent": {"id": "codex", "label": "Codex"},
  "spec": "specs/H10-design.md",
  "state": "working",
  "heartbeat_at": %q
}`, heartbeat)
	if err := os.WriteFile(filepath.Join(activityRoot, "session-1.json"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	activityStore := activity.NewStore(activityRoot)
	if err := activityStore.Refresh(); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Version: 1,
		Project: config.Project{Name: "Specview", Root: "."},
		Specs:   config.Specs{Path: "specs", Pattern: "*.md"},
		Server:  config.Server{Host: "127.0.0.1", Port: 7331},
	}
	server := NewServer(root, cfg, specStore, NewHub())
	server.SetActivityStore(activityStore)

	response := httptest.NewRecorder()
	server.index(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("board status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, want := range []string{
		"Codex",
		`specview-activity-glyph nested`,
		`specview-activity-glyph corner`,
		`specview-activity-glyph brand`,
		">CX<",
		`class="card active-spec"`,
		`data-blink="off"`,
		`data-blink="on"`,
		`specview-active-blink`,
		`>rev `,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("board does not render activity marker or control %q", want)
		}
	}
}
