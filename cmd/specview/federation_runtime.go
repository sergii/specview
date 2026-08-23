package main

import (
	"context"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/federation"
	"github.com/sergii/specview/internal/federationpeers"
	"github.com/sergii/specview/internal/federationruntime"
	"github.com/sergii/specview/internal/hoststate"
	"github.com/sergii/specview/internal/sourcecontrol"
)

func newLocalFederationBuilder(statePath, hostID string, executions hoststate.ExecutionSource) *federation.Builder {
	reader := controlplane.NewReader(statePath, executions, sourcecontrol.DefaultService())
	return federation.NewBuilder(hostID, reader)
}

func newFederationProjectionBuilder(statePath, hostID string, executions hoststate.ExecutionSource) (*federationruntime.ProjectionBuilder, error) {
	store := federationpeers.NewObservationStore(federationpeers.ObservationDir(statePath))
	return federationruntime.NewProjectionBuilder(
		newLocalFederationBuilder(statePath, hostID, executions),
		federationpeers.RegistryPath(statePath),
		store,
	)
}

func startFederationPeerRuntime(ctx context.Context, statePath string, onChange func()) error {
	store := federationpeers.NewObservationStore(federationpeers.ObservationDir(statePath))
	refresher := federationpeers.NewRefresher(nil, store)
	poller, err := federationruntime.NewPoller(
		federationpeers.RegistryPath(statePath),
		refresher,
		federationruntime.DefaultPollInterval,
		onChange,
	)
	if err != nil {
		return err
	}
	go poller.Run(ctx)
	return nil
}
