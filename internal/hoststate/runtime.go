package hoststate

import (
	"context"
	"log/slog"
	"time"
)

type Runtime struct {
	catalog  *Catalog
	scanner  Scanner
	interval time.Duration
	onChange func()
}

func NewRuntime(catalog *Catalog, scanner Scanner, interval time.Duration, onChange func()) *Runtime {
	return &Runtime{
		catalog:  catalog,
		scanner:  scanner,
		interval: interval,
		onChange: onChange,
	}
}

func (r *Runtime) Refresh() (int, error) {
	started := time.Now()
	slog.Debug("host activity refresh started")

	observations, err := r.scanner.Scan()
	if err != nil {
		slog.Error("host activity scan failed", "error", err, "duration", time.Since(started))
		return 0, err
	}
	slog.Debug("host activity scan completed",
		"observations", len(observations),
		"duration", time.Since(started),
	)

	changed, err := r.catalog.Observe(observations, time.Now())
	if err != nil {
		slog.Error("host catalog update failed", "error", err, "observations", len(observations))
		return len(observations), err
	}
	if r.onChange != nil && (changed || len(observations) > 0) {
		slog.Debug("broadcasting host activity change",
			"catalog_changed", changed,
			"observations", len(observations),
		)
		r.onChange()
	}
	slog.Debug("host activity refresh completed",
		"observations", len(observations),
		"catalog_changed", changed,
		"duration", time.Since(started),
	)
	return len(observations), nil
}

func (r *Runtime) Run(ctx context.Context) {
	slog.Info("host activity runtime started", "interval", r.interval)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	defer func() {
		if err := r.catalog.Flush(); err != nil {
			slog.Warn("flush host catalog on shutdown", "error", err)
		}
		slog.Info("host activity runtime stopped")
	}()
	for {
		select {
		case <-ctx.Done():
			slog.Debug("host activity runtime context cancelled", "error", ctx.Err())
			return
		case <-ticker.C:
			if _, err := r.Refresh(); err != nil {
				slog.Warn("refresh host activity", "error", err)
			}
		}
	}
}
