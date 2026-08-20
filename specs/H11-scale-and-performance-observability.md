---
specview:
  status: new
---

# Scale and performance observability

Measure how Specview behaves as the number of specifications grows and make performance regressions visible before optimizing the implementation.

## Scale model

Treat the following sizes as useful engineering tiers rather than promises about normal user behavior:

```text
10 specs      tiny project / smoke test
100 specs     normal larger project
1,000 specs   scale target that should still feel normal
10,000 specs  stress and benchmark dataset
```

A project with 10,000 specifications is primarily a stress case. The product should remain measurable and functional at that size even if later UI techniques such as search, filtering, or virtualization are required.

## Measure first

Instrument the current implementation before replacing components or hiding latency behind a loading screen.

Capture at least:

- total startup duration.
- config load duration.
- initial specification scan and parse duration.
- initial watcher snapshot duration.
- subsequent watcher snapshot duration.
- specification refresh duration.
- template render duration.
- HTTP request duration.
- SSE client count and refresh latency.
- process memory and goroutine count.

## Current watcher constraint

The POC watcher currently performs a complete filesystem snapshot every 250 ms. This is intentionally simple but must be benchmarked against 10, 100, 1,000, and 10,000 specifications.

If polling cost becomes material, evaluate replacing full-tree polling with filesystem notifications and narrower refreshes without changing the external Specview contract.

## Go diagnostics

Prefer built-in and low-dependency diagnostics first:

- structured timing logs with `log/slog`.
- `net/http/pprof` on a loopback-only debug endpoint when explicitly enabled.
- `runtime/metrics` for heap, GC, scheduler, goroutines, and allocation behavior.
- Go benchmarks with `-bench` and `-benchmem` for scan, parse, snapshot, and render hot paths.

Keep diagnostics opt-in where they expose internal profiling endpoints.

## Optional observability lab

After native measurements are useful, provide an optional local lab for OpenTelemetry export and a full observability backend such as SigNoz. The observability stack must not become a runtime dependency of the Specview binary.

## Startup UX

Only after startup phases are measured, decide whether the web server should start before repository scanning and render a temporary Specview logo / preparing state while the initial projection is built asynchronously.

A loading screen must communicate real startup state, not mask an unexplained performance regression.

## Acceptance criteria

- reproducible datasets exist for 10, 100, 1,000, and 10,000 specifications.
- benchmarks report time and allocations for the main filesystem and rendering paths.
- startup phase timings make it possible to identify where startup time is spent.
- polling overhead is measurable independently from Markdown parsing and rendering.
- optional profiling binds only to loopback and is disabled by default.
- 1,000 specifications is treated as the primary comfortable scale target.
- 10,000 specifications is treated as a stress target rather than assumed normal product usage.
