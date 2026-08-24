package controlplane

import (
	"context"

	"github.com/sergii/specview/internal/executionhistory"
	"github.com/sergii/specview/internal/hoststate"
)

func (r *Reader) GetExecutionHistory(context.Context) (executionhistory.Projection, error) {
	catalog, err := hoststate.OpenCatalog(r.statePath)
	if err != nil {
		return executionhistory.Projection{}, err
	}
	return executionhistory.Build(catalog.Hostname(), catalog.Repositories()), nil
}
