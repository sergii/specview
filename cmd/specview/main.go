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

	"github.com/sergii/specview/demo"
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
		return initProject(args[1:])
	case "demo":
		return serveDemo()
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

func initProject(args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(args) == 1 && args[0] == "--demo" {
		created, err := demo.Create(root)
		if err != nil {
			return err
		}
		fmt.Println("✓ Created .specview.yaml for Demo Project")
		fmt.Printf("✓ Created %d demo specifications in specs/\n", created)
		fmt.Println("\nRun 'specview' to start observing demo specifications.")
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("unknown init option %q; supported option: --demo", args[0])
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

func serveDemo() error {
	root, err := os.MkdirTemp("", "specview-demo-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	created, err := demo.Create(root)
	if err != nil {
		return err
	}
	fmt.Printf("Specview Demo Project · %d demo specs\n", created)
	return serveRoot(root)
}

func serve() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	return serveRoot(root)
}

func serveRoot(root string) error {
	cfg, err := config.Load(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New(".specview.yaml not found; run 'specview init' first")
		}
		return err
	}

	specRoot := filepath.Join(root, cfg.Specs.Path)
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := webui.NewServer(root, cfg, store, hub)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	projectName := cfg.Project.Name
	if projectName == "" {
		projectName = filepath.Base(root)
	}
	fmt.Printf("Specview watching %s · %s\n", projectName, cfg.Specs.Path)
	fmt.Printf("http://%s\n", addr)

	return server.ListenAndServe(ctx)
}

func printHelp() {
	fmt.Printf(`Specview - live, read-only observation for Markdown specifications.

Usage:
  specview                 Start the dashboard in the current repository
  specview serve           Start the dashboard in the current repository
  specview init            Create .specview.yaml and specs/
  specview init --demo     Initialize the current repository with 10 demo specs
  specview demo            Run an isolated Demo Project without changing the current repository
  specview version         Print the version
  specview help            Show this help
`)
}
