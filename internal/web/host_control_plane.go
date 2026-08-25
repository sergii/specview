package web

import (
	"context"

	"github.com/sergii/specview/internal/controlplane"
	"github.com/sergii/specview/internal/hoststate"
)

type hostControlPlaneSummary = controlplane.GetHostControlPlaneResult

func (s *HostServer) hostControlPlane(ctx context.Context, repositories []hoststate.Repository) hostControlPlaneSummary {
	return controlplane.BuildHostControlPlane(ctx, s.catalog.Hostname(), repositories, s.sourceControl)
}
