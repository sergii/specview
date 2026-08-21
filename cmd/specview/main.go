package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sergii/specview/internal/config"
	"github.com/sergii/specview/internal/hoststate"
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
	case "doctor":
		return doctor()
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
	fmt.Println("\nRun 'specview' to start observing this host.")
	return nil
}

func doctor() error {
	diagnostics, err := hoststate.DiagnoseCodex()
	if err != nil {
		return fmt.Errorf("diagnose Codex discovery: %w", err)
	}

	fmt.Println("Specview doctor - Codex discovery")
	if len(diagnostics) == 0 {
		fmt.Println("Matched Codex processes: 0")
		return nil
	}
	fmt.Printf("Matched Codex processes: %d\n", len(diagnostics))
	for _, diagnostic := range diagnostics {
		fmt.Printf("\nPID %d\n", diagnostic.PID)
		if diagnostic.Command != "" {
			fmt.Printf("  command: %s\n", diagnostic.Command)
		}
		fmt.Printf("  matched: %t\n", diagnostic.Matched)
		if diagnostic.CWD != "" {
			fmt.Printf("  cwd: %s\n", diagnostic.CWD)
		}
		if diagnostic.RepositoryRoot != "" {
			fmt.Printf("  repository: %s\n", diagnostic.RepositoryRoot)
		}
		fmt.Printf("  stage: %s\n", diagnostic.Stage)
		if diagnostic.Error != "" {
			fmt.Printf("  error: %s\n", diagnostic.Error)
		}
	}
	return nil
}

func serve() error {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return err
	}
	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		return err
	}

	hub := webui.NewHub()
	runtime := hoststate.NewRuntime(catalog, hoststate.NewCodexScanner(), 2*time.Second, hub.Broadcast)
	if _, err := runtime.Refresh(); err != nil {
		slog.Warn("initial host activity scan failed", "error", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go runtime.Run(ctx)

	const host = "127.0.0.1"
	const port = 7331
	server := webui.NewHostServer(catalog, hub, host, port)
	fmt.Printf("Specview observing host %s\n", catalog.Hostname())
	fmt.Printf("State: %s\n", statePath)
	fmt.Printf("http://%s:%d\n", host, port)
	return server.ListenAndServe(ctx)
}

func printHelp() {
	fmt.Printf(`Specview - local-first observation for repo-native, spec-driven software work.

Usage:
  specview              Start the host dashboard and observe active repositories
  specview serve        Start the host dashboard and observe active repositories
  specview init         Detect the current repository convention and create .specview.yaml
  specview doctor       Diagnose host Codex process -> cwd -> Git repository discovery
  specview version      Print the version
  specview help         Show this help

The host dashboard does not require .specview.yaml. Project configuration remains
an optional repository-level override.
`)
}
