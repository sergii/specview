package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
	"github.com/sergii/specview/internal/web"
)

const fixtureHead = "abc123"

type fixtureSourceControl struct {
	root string
}

func (s fixtureSourceControl) Inspect(context.Context, string) (sourcecontrol.RepositoryContext, error) {
	return sourcecontrol.RepositoryContext{
		Git: sourcecontrol.GitContext{
			Remote: "https://github.com/sergii/specview.git",
			Worktrees: []sourcecontrol.Worktree{{
				Path:       s.root,
				Branch:     "feat/acceptance-policy",
				Head:       fixtureHead,
				DirtyCount: 0,
			}},
		},
	}, nil
}

func main() {
	root := filepath.Join(os.TempDir(), "specview-e2e", "repository")
	if err := os.RemoveAll(filepath.Dir(root)); err != nil {
		log.Fatal(err)
	}
	if err := writeFixture(root); err != nil {
		log.Fatal(err)
	}

	catalog, err := hoststate.OpenCatalog("")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := catalog.Observe([]hoststate.Observation{{
		Agent:          "Codex",
		PID:            4242,
		RepositoryRoot: root,
	}}, time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	server := web.NewHostServerWithSources(
		catalog,
		web.NewHub(),
		"127.0.0.1",
		7332,
		nil,
		fixtureSourceControl{root: root},
	)
	log.Printf("Specview e2e fixture server listening on http://127.0.0.1:7332")
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}

func writeFixture(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".specview", "evidence"), 0o755); err != nil {
		return err
	}

	config := `version: 1
project:
  name: "Specview E2E"
  root: "."
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
    - lint
server:
  host: 127.0.0.1
  port: 7332
`
	if err := os.WriteFile(filepath.Join(root, ".specview.yaml"), []byte(config), 0o644); err != nil {
		return err
	}

	spec := `---
specview:
  status: in_progress
---
# H17 Acceptance Policy

This deterministic fixture exists only for browser conformance tests.
`
	if err := os.WriteFile(filepath.Join(root, "specs", "H17.md"), []byte(spec), 0o644); err != nil {
		return err
	}

	for index, check := range []string{"unit-tests", "lint"} {
		observed := fmt.Sprintf("2026-08-23T12:00:0%dZ", index)
		record := fmt.Sprintf(`{
  "version": 1,
  "id": "H17-%s-e2e",
  "work_item_id": "H17",
  "revision": "git:%s",
  "check": %q,
  "kind": "test",
  "provider": "e2e-fixture",
  "result": "passed",
  "finished_at": %q,
  "observed_at": %q,
  "summary": "fixture passed"
}
`, check, fixtureHead, check, observed, observed)
		if err := os.WriteFile(filepath.Join(root, ".specview", "evidence", check+".json"), []byte(record), 0o644); err != nil {
			return err
		}
	}
	return nil
}
