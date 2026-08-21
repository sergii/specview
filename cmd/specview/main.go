package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "serve":
		return serve()
	case "init":
		return initProject()
	case "version", "--version", "-v":
		fmt.Printf("Specview %s\n", version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun 'specview help' for usage", command)
	}
}

func initProject() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	createdConfig, createdArtifactRoot, artifactPath, err := config.Init(root)
	if err != nil {
		return err
	}

	if createdConfig {
		fmt.Println("✓ Created .specview.yaml")
	} else {
		fmt.Println("• .specview.yaml already exists")
	}
	if createdArtifactRoot {
		fmt.Printf("✓ Created %s/\n", artifactPath)
	} else {
		fmt.Printf("• %s/ already exists\n", artifactPath)
	}
	fmt.Println("\nRun 'specview' to start observing specifications.")
	return nil
}

func serve() error {
	configRoot, err := os.Getwd()
	if err != nil {
		return err
	}
	return serveRoot(configRoot)
}

func serveRoot(configRoot string) error {
	cfg, err := config.Load(configRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New(".specview.yaml not found; run 'specview init' first")
		}
		return err
	}

	projectRoot := cfg.ResolveProjectRoot(configRoot)
	info, err := os.Stat(projectRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("project.root %q does not exist", cfg.Project.Root)
		}
		return fmt.Errorf("stat project.root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project.root %q is not a directory", cfg.Project.Root)
	}

	specRoot := filepath.Join(projectRoot, cfg.Specs.Path)
	if err := os.MkdirAll(specRoot, 0o755); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	adapter, err := specs.NewAdapter(cfg.Specs.Adapter, specRoot, cfg.Specs.Pattern)
	if err != nil {
		return err
	}
	store := specs.NewStoreWithAdapter(adapter)
	if err := store.Refresh(); err != nil {
		return err
	}

	hub := webui.NewHub()
	refresh := func() {
		if err := store.Refresh(); err != nil {
			slog.Error("refresh specifications", "error", err)
			return
		}
		hub.Broadcast()
	}

	var watchers []*watch.Watcher
	for _, root := range adapter.WatchRoots() {
		watcher, err := watch.New(root, refresh)
		if err != nil {
			for _, started := range watchers {
				_ = started.Close()
			}
			return fmt.Errorf("start watcher for %s: %w", root, err)
		}
		watchers = append(watchers, watcher)
	}
	defer func() {
		for _, watcher := range watchers {
			_ = watcher.Close()
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := webui.NewServer(projectRoot, cfg, store, hub)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	projectName := cfg.Project.Name
	if projectName == "" {
		projectName = filepath.Base(projectRoot)
	}
	observedPath := cfg.Specs.Path
	if cfg.Project.Root != "." {
		observedPath = filepath.Join(cfg.Project.Root, cfg.Specs.Path)
	}
	fmt.Printf("Specview watching %s · %s · adapter=%s\n", projectName, observedPath, store.AdapterName())
	fmt.Printf("http://%s\n", addr)

	return server.ListenAndServe(ctx)
}

func printHelp() {
	fmt.Printf(`Specview - live, read-only observation for repo-native specifications.

Usage:
  specview              Start the dashboard using .specview.yaml in the current directory
  specview serve        Start the dashboard using .specview.yaml in the current directory
  specview init         Detect the repository convention and create .specview.yaml
  specview version      Print the version
  specview help         Show this help
`)
}
