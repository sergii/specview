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
	"github.com/sergii/specview/internal/logging"
	webui "github.com/sergii/specview/internal/web"
)

var version = "dev"

func main() {
	_, settings := logging.Configure(version)
	slog.Debug("process started",
		"pid", os.Getpid(),
		"args", os.Args[1:],
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
	slog.Info("doctor started", "check", "codex-discovery")
	diagnostics, err := hoststate.DiagnoseCodex()
	if err != nil {
		return fmt.Errorf("diagnose Codex discovery: %w", err)
	}

	fmt.Println("Specview doctor - Codex discovery")
	if len(diagnostics) == 0 {
		fmt.Println("Matched Codex processes: 0")
		slog.Warn("doctor found no Codex processes")
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
		slog.Debug("doctor diagnostic",
			"pid", diagnostic.PID,
			"matched", diagnostic.Matched,
			"cwd", diagnostic.CWD,
			"repository", diagnostic.RepositoryRoot,
			"stage", diagnostic.Stage,
			"error", diagnostic.Error,
		)
	}
	slog.Info("doctor completed", "diagnostics", len(diagnostics))
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

	hub := webui.NewHub()
	runtime := hoststate.NewRuntime(catalog, hoststate.NewCodexScanner(), 2*time.Second, hub.Broadcast)
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
	server := webui.NewHostServer(catalog, hub, host, port)
	slog.Info("Specview host observer started",
		"hostname", catalog.Hostname(),
		"state", statePath,
		"address", fmt.Sprintf("http://%s:%d", host, port),
		"scan_interval", 2*time.Second,
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
  specview doctor       Diagnose host Codex process -> cwd -> Git repository discovery
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
