package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sergii/specview/internal/activity"
	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/specs"
	"github.com/sergii/specview/internal/watch"
	webui "github.com/sergii/specview/internal/web"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "specview: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return serve(nil)
	}

	switch args[0] {
	case "serve":
		return serve(args[1:])
	case "init":
		return initProject()
	case "version", "--version", "-v":
		fmt.Printf("Specview %s\n", version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun 'specview help' for usage", args[0])
	}
}

func initProject() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	createdConfig, createdSpecs, err := config.Init(root)
	if err != nil {
		return err
	}

	if createdConfig {
		fmt.Println("✓ Created .specview.yaml")
	} else {
		fmt.Println("• .specview.yaml already exists")
	}
	if createdSpecs {
		fmt.Println("✓ Created specs/")
	} else {
		fmt.Println("• specs/ already exists")
	}
	fmt.Println("\nRun 'specview' to start observing specifications.")
	return nil
}

func serve(inputs []string) error {
	var configRoots []string
	var err error
	if len(inputs) == 0 {
		root, cwdErr := os.Getwd()
		if cwdErr != nil {
			return cwdErr
		}
		configRoots = []string{root}
	} else {
		configRoots, err = discoverConfigRoots(inputs)
		if err != nil {
			return err
		}
		if len(configRoots) == 0 {
			return errors.New("no .specview.yaml projects found in the supplied paths")
		}
	}
	return serveConfigRoots(configRoots)
}

type projectRuntime struct {
	source          webui.ProjectSource
	specWatcher     *watch.Watcher
	activityWatcher *watch.Watcher
}

func (p *projectRuntime) Close() {
	if p.activityWatcher != nil {
		_ = p.activityWatcher.Close()
	}
	if p.specWatcher != nil {
		_ = p.specWatcher.Close()
	}
}

func serveConfigRoots(configRoots []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hub := webui.NewHub()
	runtimes := make([]*projectRuntime, 0, len(configRoots))
	for _, configRoot := range configRoots {
		runtime, err := openProject(configRoot, hub, ctx)
		if err != nil {
			for _, opened := range runtimes {
				opened.Close()
			}
			return fmt.Errorf("open %s: %w", configRoot, err)
		}
		runtimes = append(runtimes, runtime)
	}
	defer func() {
		for _, runtime := range runtimes {
			runtime.Close()
		}
	}()

	sources := make([]webui.ProjectSource, 0, len(runtimes))
	for _, runtime := range runtimes {
		sources = append(sources, runtime.source)
	}
	serverCfg := sources[0].Config.Server
	server := webui.NewWorkspaceServer(sources, serverCfg, hub)
	addr := fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port)

	if len(sources) == 1 {
		source := sources[0]
		projectName := source.Config.Project.Name
		if projectName == "" {
			projectName = filepath.Base(source.Root)
		}
		observedPath := source.Config.Specs.Path
		if source.Config.Project.Root != "." {
			observedPath = filepath.Join(source.Config.Project.Root, source.Config.Specs.Path)
		}
		fmt.Printf("Specview watching %s · %s\n", projectName, observedPath)
	} else {
		fmt.Printf("Specview workspace · %d projects\n", len(sources))
		for _, source := range sources {
			fmt.Printf("  %s\n", filepath.ToSlash(source.Root))
		}
	}
	fmt.Printf("http://%s\n", addr)

	return server.ListenAndServe(ctx)
}

func openProject(configRoot string, hub *webui.Hub, ctx context.Context) (*projectRuntime, error) {
	cfg, err := config.Load(configRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New(".specview.yaml not found; run 'specview init' first")
		}
		return nil, err
	}

	projectRoot := cfg.ResolveProjectRoot(configRoot)
	info, err := os.Stat(projectRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("project.root %q does not exist", cfg.Project.Root)
		}
		return nil, fmt.Errorf("stat project.root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project.root %q is not a directory", cfg.Project.Root)
	}

	specRoot := filepath.Join(projectRoot, cfg.Specs.Path)
	if err := os.MkdirAll(specRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create specs directory: %w", err)
	}
	store := specs.NewStore(specRoot, cfg.Specs.Pattern)
	if err := store.Refresh(); err != nil {
		return nil, err
	}

	specWatcher, err := watch.New(specRoot, func() {
		if err := store.Refresh(); err != nil {
			slog.Error("refresh specifications", "project", projectRoot, "error", err)
			return
		}
		hub.Broadcast()
	})
	if err != nil {
		return nil, fmt.Errorf("start watcher: %w", err)
	}

	activityRoot := filepath.Join(projectRoot, activity.RelativeDir)
	activityStore := activity.NewStore(activityRoot)
	if err := activityStore.Refresh(); err != nil {
		_ = specWatcher.Close()
		return nil, err
	}
	for _, parseErr := range activityStore.Errors() {
		slog.Warn("invalid agent activity", "project", projectRoot, "path", parseErr.Path, "error", parseErr.Message)
	}

	var signatureMu sync.Mutex
	signature := activityStore.ActiveSignature(time.Now())
	broadcastIfChanged := func(now time.Time) {
		current := activityStore.ActiveSignature(now)
		signatureMu.Lock()
		changed := current != signature
		if changed {
			signature = current
		}
		signatureMu.Unlock()
		if changed {
			hub.Broadcast()
		}
	}

	activityWatcher, err := watch.New(activityRoot, func() {
		if err := activityStore.Refresh(); err != nil {
			slog.Error("refresh agent activity", "project", projectRoot, "error", err)
			return
		}
		for _, parseErr := range activityStore.Errors() {
			slog.Warn("invalid agent activity", "project", projectRoot, "path", parseErr.Path, "error", parseErr.Message)
		}
		broadcastIfChanged(time.Now())
	})
	if err != nil {
		_ = specWatcher.Close()
		return nil, fmt.Errorf("start activity watcher: %w", err)
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				broadcastIfChanged(now)
			}
		}
	}()

	return &projectRuntime{
		source:          webui.NewProjectSource(projectRoot, cfg, store, activityStore),
		specWatcher:     specWatcher,
		activityWatcher: activityWatcher,
	}, nil
}

func discoverConfigRoots(inputs []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, input := range inputs {
		path, err := expandPath(input)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", input, err)
		}

		if !info.IsDir() {
			if filepath.Base(path) != config.FileName {
				return nil, fmt.Errorf("%s is not a directory or %s", input, config.FileName)
			}
			seen[filepath.Dir(path)] = struct{}{}
			continue
		}

		if hasConfig(path) {
			seen[path] = struct{}{}
			continue
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", input, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			child := filepath.Join(path, entry.Name())
			if hasConfig(child) {
				seen[child] = struct{}{}
			}
		}
	}

	roots := make([]string, 0, len(seen))
	for root := range seen {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		roots = append(roots, filepath.Clean(abs))
	}
	sort.Strings(roots)
	return roots, nil
}

func hasConfig(root string) bool {
	info, err := os.Stat(filepath.Join(root, config.FileName))
	return err == nil && !info.IsDir()
}

func expandPath(input string) (string, error) {
	if input == "~" || strings.HasPrefix(input, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if input == "~" {
			input = home
		} else {
			input = filepath.Join(home, strings.TrimPrefix(input, "~/"))
		}
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func printHelp() {
	fmt.Printf(`Specview - live, read-only observation for Markdown specifications.

Usage:
  specview                         Start the current project dashboard
  specview serve                   Start the current project dashboard
  specview serve PATH [PATH ...]   Discover projects and start a workspace dashboard
  specview init                    Create .specview.yaml and specs/
  specview version                 Print the version
  specview help                    Show this help

Workspace discovery treats a directory containing .specview.yaml as one project.
If a supplied directory has no .specview.yaml, Specview discovers configured
projects in its immediate child directories.
`)
}
