package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/identity"
	"github.com/sergii/specview/internal/mcpserver"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func serveMCP() error {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return err
	}
	hostIdentity, err := identity.LoadOrCreateHostForCatalog(statePath)
	if err != nil {
		return err
	}

	executions := hoststate.DefaultExecutionRegistry()
	reader := controlplane.NewReader(
		statePath,
		executions,
		sourcecontrol.DefaultService(),
	)
	federationReader, err := newFederationProjectionBuilder(statePath, hostIdentity.ID, executions)
	if err != nil {
		return err
	}
	server := mcpserver.NewWithFederation(reader, federationReader, version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.Serve(ctx, os.Stdin, os.Stdout)
}
