package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationhttp"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/identity"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func runFederation(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("federation command requires snapshot, aggregate, status, serve, pull, or peer")
	}

	switch args[0] {
	case "snapshot":
		if len(args) != 1 {
			return fmt.Errorf("usage: specview federation snapshot")
		}
		return writeFederationSnapshot(context.Background(), os.Stdout)
	case "aggregate":
		if len(args) < 2 {
			return fmt.Errorf("usage: specview federation aggregate <snapshot.json>...")
		}
		return writeFederationProjection(args[1:], os.Stdout)
	case "status":
		if len(args) != 1 {
			return fmt.Errorf("usage: specview federation status")
		}
		return writeFederationStatus(context.Background(), os.Stdout)
	case "serve":
		if len(args) != 1 {
			return fmt.Errorf("usage: specview federation serve")
		}
		return serveFederationHTTP()
	case "pull":
		return pullFederationSnapshot(args[1:], os.Stdout)
	case "peer":
		return runFederationPeer(args[1:])
	default:
		return fmt.Errorf("unknown federation command %q", args[0])
	}
}

func localFederationBuilder() (*federation.Builder, error) {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return nil, err
	}
	hostIdentity, err := identity.LoadOrCreateHostForCatalog(statePath)
	if err != nil {
		return nil, err
	}
	reader := controlplane.NewReader(
		statePath,
		hoststate.DefaultExecutionRegistry(),
		sourcecontrol.DefaultService(),
	)
	return federation.NewBuilder(hostIdentity.ID, reader), nil
}

func writeFederationSnapshot(ctx context.Context, destination io.Writer) error {
	builder, err := localFederationBuilder()
	if err != nil {
		return err
	}
	snapshot, err := builder.Build(ctx)
	if err != nil {
		return err
	}
	return writeFederationJSON(destination, snapshot)
}

func writeFederationStatus(ctx context.Context, destination io.Writer) error {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return err
	}
	builder, err := localFederationBuilder()
	if err != nil {
		return err
	}
	store := federationpeers.NewObservationStore(federationpeers.ObservationDir(statePath))
	projectionBuilder, err := federationruntime.NewProjectionBuilder(
		builder,
		federationpeers.RegistryPath(statePath),
		store,
	)
	if err != nil {
		return err
	}
	projection, err := projectionBuilder.Build(ctx)
	if err != nil {
		return err
	}
	return writeFederationJSON(destination, projection)
}

func serveFederationHTTP() error {
	builder, err := localFederationBuilder()
	if err != nil {
		return err
	}
	handler, err := federationhttp.NewHandler(builder)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Addr:              federationhttp.DefaultAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	fmt.Printf("Specview federation snapshot endpoint on http://%s%s\n", federationhttp.DefaultAddress, federationhttp.SnapshotPath)
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func pullFederationSnapshot(args []string, destination io.Writer) error {
	rawURL, expectedHostID, err := parseFederationPullArgs(args)
	if err != nil {
		return err
	}
	snapshot, err := federationhttp.NewClient().Fetch(context.Background(), rawURL, expectedHostID)
	if err != nil {
		return err
	}
	return writeFederationJSON(destination, snapshot)
}

func parseFederationPullArgs(args []string) (rawURL, expectedHostID string, err error) {
	const usage = "usage: specview federation pull [--expect-host host:<uuid>] <url>"
	sawExpectedHost := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--expect-host":
			if sawExpectedHost || i+1 >= len(args) {
				return "", "", errors.New(usage)
			}
			sawExpectedHost = true
			i++
			expectedHostID = strings.TrimSpace(args[i])
			if expectedHostID == "" {
				return "", "", errors.New(usage)
			}
		case strings.HasPrefix(arg, "--expect-host="):
			if sawExpectedHost {
				return "", "", errors.New(usage)
			}
			sawExpectedHost = true
			expectedHostID = strings.TrimSpace(strings.TrimPrefix(arg, "--expect-host="))
			if expectedHostID == "" {
				return "", "", errors.New(usage)
			}
		default:
			if strings.HasPrefix(arg, "-") || rawURL != "" {
				return "", "", errors.New(usage)
			}
			rawURL = strings.TrimSpace(arg)
		}
	}
	if rawURL == "" {
		return "", "", errors.New(usage)
	}
	return rawURL, expectedHostID, nil
}

func writeFederationProjection(paths []string, destination io.Writer) error {
	snapshots := make([]federation.HostSnapshot, 0, len(paths))
	for _, path := range paths {
		data, err := readSnapshotInput(path)
		if err != nil {
			return err
		}
		snapshot, err := federation.DecodeSnapshot(data)
		if err != nil {
			return fmt.Errorf("decode federation snapshot %s: %w", path, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	projection, err := federation.NewAggregator().Aggregate(snapshots...)
	if err != nil {
		return err
	}
	return writeFederationJSON(destination, projection)
}

func readSnapshotInput(path string) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read federation snapshot from stdin: %w", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read federation snapshot %s: %w", path, err)
	}
	return data, nil
}

func writeFederationJSON(destination io.Writer, value any) error {
	encoder := json.NewEncoder(destination)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode federation JSON: %w", err)
	}
	return nil
}
