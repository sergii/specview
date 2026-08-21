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
	observations, err := r.scanner.Scan()
	if err != nil {
		return 0, err
	}
	changed, err := r.catalog.Observe(observations, time.Now())
	if err != nil {
		return len(observations), err
	}
	if r.onChange != nil && (changed || len(observations) > 0) {
		r.onChange()
	}
	return len(observations), nil
}

func (r *Runtime) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.Refresh(); err != nil {
				slog.Warn("refresh host activity", "error", err)
			}
		}
	}
}
