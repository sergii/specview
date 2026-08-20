package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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
		return fmt.Errorf("create specs directory: %w", err)
	}

	store := specs.NewStore(specRoot, cfg.Specs.Pattern)
	if err := store.Refresh(); err != nil {
		return err
	}

	hub := webui.NewHub()
	watcher, err := watch.New(specRoot, func() {
		if err := store.Refresh(); err != nil {
			slog.Error("refresh specifications", "error", err)
			return
		}
		hub.Broadcast()
	})
	if err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}
	defer watcher.Close()

	activityRoot := filepath.Join(projectRoot, activity.RelativeDir)
	activityStore := activity.NewStore(activityRoot)
	if err := activityStore.Refresh(); err != nil {
		return err
	}
	for _, parseErr := range activityStore.Errors() {
		slog.Warn("invalid agent activity", "path", parseErr.Path, "error", parseErr.Message)
	}

	var activitySignatureMu sync.Mutex
	activitySignature := activityStore.ActiveSignature(time.Now())
	broadcastActivityIfChanged := func(now time.Time) {
		current := activityStore.ActiveSignature(now)
		activitySignatureMu.Lock()
		changed := current != activitySignature
		if changed {
			activitySignature = current
		}
		activitySignatureMu.Unlock()
		if changed {
			hub.Broadcast()
		}
	}

	activityWatcher, err := watch.New(activityRoot, func() {
		if err := activityStore.Refresh(); err != nil {
			slog.Error("refresh agent activity", "error", err)
			return
		}
		for _, parseErr := range activityStore.Errors() {
			slog.Warn("invalid agent activity", "path", parseErr.Path, "error", parseErr.Message)
		}
		broadcastActivityIfChanged(time.Now())
	})
	if err != nil {
		return fmt.Errorf("start activity watcher: %w", err)
	}
	defer activityWatcher.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				broadcastActivityIfChanged(now)
			}
		}
	}()

	server := webui.NewServer(projectRoot, cfg, store, hub)
	server.SetActivityStore(activityStore)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	projectName := cfg.Project.Name
	if projectName == "" {
		projectName = filepath.Base(projectRoot)
	}
	observedPath := cfg.Specs.Path
	if cfg.Project.Root != "." {
		observedPath = filepath.Join(cfg.Project.Root, cfg.Specs.Path)
	}
	fmt.Printf("Specview watching %s · %s\n", projectName, observedPath)
	fmt.Printf("http://%s\n", addr)

	return server.ListenAndServe(ctx)
}

func printHelp() {
	fmt.Printf(`Specview - live, read-only observation for Markdown specifications.

Usage:
  specview              Start the dashboard using .specview.yaml in the current directory
  specview serve        Start the dashboard using .specview.yaml in the current directory
  specview init         Create .specview.yaml and specs/
  specview version      Print the version
  specview help         Show this help
`)
}
