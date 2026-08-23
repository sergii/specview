package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/mcpserver"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func serveMCP() error {
	statePath, err := hoststate.DefaultStatePath()
	if err != nil {
		return err
	}

	reader := controlplane.NewReader(
		statePath,
		hoststate.DefaultExecutionRegistry(),
		sourcecontrol.DefaultService(),
	)
	server := mcpserver.New(reader, version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.Serve(ctx, os.Stdin, os.Stdout)
}
