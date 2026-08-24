package hoststate

import (
	"context"
	"log/slog"
	"time"
)

type Runtime struct {
	catalog    *Catalog
	scanner    Scanner
	executions ExecutionSource
	interval   time.Duration
	onChange   func()
}

func NewRuntime(catalog *Catalog, scanner Scanner, interval time.Duration, onChange func()) *Runtime {
	runtime := &Runtime{
		catalog:  catalog,
		scanner:  scanner,
		interval: interval,
		onChange: onChange,
	}
	if source, ok := scanner.(ExecutionSource); ok {
		runtime.executions = source
	}
	return runtime
}

func (r *Runtime) Refresh() (int, error) {
	started := time.Now()
	slog.Debug("host activity refresh started")

	if r.executions != nil {
		sessions, err := r.executions.Sessions()
		if err != nil {
			slog.Error("logical execution scan failed", "error", err, "duration", time.Since(started))
			return 0, err
		}
		slog.Debug("logical execution scan completed",
			"sessions", len(sessions),
			"duration", time.Since(started),
		)
		changed, err := r.catalog.ObserveExecutions(sessions, time.Now())
		if err != nil {
			slog.Error("host catalog logical execution update failed", "error", err, "sessions", len(sessions))
			return len(sessions), err
		}
		r.broadcastChange(changed, len(sessions), "sessions")
		slog.Debug("host activity refresh completed",
			"sessions", len(sessions),
			"catalog_changed", changed,
			"duration", time.Since(started),
		)
		return len(sessions), nil
	}

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
	r.broadcastChange(changed, len(observations), "observations")
	slog.Debug("host activity refresh completed",
		"observations", len(observations),
		"catalog_changed", changed,
		"duration", time.Since(started),
	)
	return len(observations), nil
}

func (r *Runtime) broadcastChange(changed bool, count int, unit string) {
	if r.onChange == nil || (!changed && count == 0) {
		return
	}
	slog.Debug("broadcasting host activity change",
		"catalog_changed", changed,
		unit, count,
	)
	r.onChange()
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
