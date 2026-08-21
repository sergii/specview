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
	"github.com/sergii/specview/internal/hostindex"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/logging"
	webui "github.com/sergii/specview/internal/web"
)

var version = "dev"

func main() {
	_, settings := logging.Configure(version)
	slog.Debug("process started",
		"pid", os.Getpid(),
		"arg_count", len(os.Args)-1,
		"log_format", settings.Format,
	)

	if err := run(os.Args[1:]); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}
	slog.Debug("command selected", "command", command)

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
	slog.Info("initializing project", "root", root)

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
	slog.Info("project initialization completed",
		"root", root,
		"created_config", createdConfig,
		"created_artifact_root", createdArtifactRoot,
		"artifact_path", artifactPath,
	)
	return nil
}

func doctor() error {
	registry := hoststate.DefaultExecutionRegistry()
	adapter, ok := registry.Adapter("codex")
	if !ok {
		return fmt.Errorf("Codex execution adapter is not registered")
	}

	slog.Info("doctor started", "adapter", adapter.Name(), "check", "process-cwd-repository")
	diagnostics, err := adapter.Diagnostics()
	if err != nil {
		return fmt.Errorf("diagnose %s execution adapter: %w", adapter.Name(), err)
	}

	fmt.Printf("Specview doctor - %s execution adapter\n", adapter.Name())
	if len(diagnostics) == 0 {
		fmt.Println("Matched processes: 0")
		slog.Warn("doctor found no matching processes", "adapter", adapter.Name())
		return nil
	}
	fmt.Printf("Matched processes: %d\n", len(diagnostics))
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
		slog.Debug("doctor diagnostic",
			"adapter", adapter.Name(),
			"pid", diagnostic.PID,
			"matched", diagnostic.Matched,
			"cwd", diagnostic.CWD,
			"repository", diagnostic.RepositoryRoot,
			"stage", diagnostic.Stage,
			"error", diagnostic.Error,
		)
	}
	slog.Info("doctor completed", "adapter", adapter.Name(), "diagnostics", len(diagnostics))
	return nil
}

func serve() error {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return err
	}
	slog.Debug("resolved host state path", "path", statePath)

	catalog, err := hoststate.OpenCatalog(statePath)
	if err != nil {
		return err
	}

	var index *hostindex.Index
	indexPath := hostindex.DefaultPath(statePath)
	if indexPath != "" {
		index, err = hostindex.Open(indexPath)
		if err != nil {
			slog.Warn("SQLite host index unavailable", "path", indexPath, "error", err)
			index = nil
		} else {
			defer index.Close()
			if err := index.Sync(context.Background(), catalog.Repositories()); err != nil {
				slog.Warn("initial SQLite host index sync failed", "path", indexPath, "error", err)
			} else {
				slog.Info("SQLite host index ready", "path", indexPath)
			}
		}
	}

	executions := hoststate.DefaultExecutionRegistry()
	hub := webui.NewHub()
	onChange := func() {
		if index != nil {
			if err := index.Sync(context.Background(), catalog.Repositories()); err != nil {
				slog.Warn("SQLite host index sync failed", "path", index.Path(), "error", err)
			}
		}
		hub.Broadcast()
	}
	runtime := hoststate.NewRuntime(catalog, executions, 2*time.Second, onChange)
	observations, err := runtime.Refresh()
	if err != nil {
		slog.Warn("initial host activity scan failed", "error", err)
	} else {
		slog.Info("initial host activity scan completed", "observations", observations)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go runtime.Run(ctx)

	const host = "127.0.0.1"
	const port = 7331
	var repositorySearch webui.RepositorySearcher
	if index != nil {
		repositorySearch = index
	}
	server := webui.NewHostServerWithSearch(catalog, hub, host, port, executions, repositorySearch)
	slog.Info("Specview host observer started",
		"hostname", catalog.Hostname(),
		"state", statePath,
		"index", indexPath,
		"address", fmt.Sprintf("http://%s:%d", host, port),
		"scan_interval", 2*time.Second,
		"execution_registry", "default",
	)

	err = server.ListenAndServe(ctx)
	if err == nil {
		slog.Info("Specview stopped cleanly")
	}
	return err
}

func printHelp() {
	fmt.Printf(`Specview - local-first observation for repo-native, spec-driven software work.

Usage:
  specview              Start the host dashboard and observe active repositories
  specview serve        Start the host dashboard and observe active repositories
  specview init         Detect the current repository convention and create .specview.yaml
  specview doctor       Diagnose the registered Codex execution adapter
  specview version      Print the version
  specview help         Show this help

Logging:
  SPECVIEW_LOG_FORMAT   console (default) or json
  SPECVIEW_LOG_LEVEL    debug (dev default), info, warn, or error
  SPECVIEW_LOG_SOURCE   true/false; source is on by default in console mode
  SPECVIEW_LOG_COLOR    true/false; colors are on by default in console mode
  SPECVIEW_ENV          production defaults logging to JSON/info

The host dashboard does not require .specview.yaml. Project configuration remains
an optional repository-level override.
`)
}
