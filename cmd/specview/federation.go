package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/identity"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func runFederation(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("federation command requires snapshot or aggregate")
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
	default:
		return fmt.Errorf("unknown federation command %q", args[0])
	}
}

func writeFederationSnapshot(ctx context.Context, destination io.Writer) error {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return err
	}
	hostIdentity, err := identity.LoadOrCreateHostForCatalog(statePath)
	if err != nil {
		return err
	}
	reader := controlplane.NewReader(
		statePath,
		hoststate.DefaultExecutionRegistry(),
		sourcecontrol.DefaultService(),
	)
	snapshot, err := federation.NewBuilder(hostIdentity.ID, reader).Build(ctx)
	if err != nil {
		return err
	}
	return writeFederationJSON(destination, snapshot)
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
