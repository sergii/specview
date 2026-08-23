---
specview:
  status: in_progress
---

# H23 - Federation Runtime and Multi-Host Projection

## Goal

Keep configured federation peers current while `specview serve` is running and expose one deterministic read-only multi-host projection without changing H20-H22 contracts.

## Runtime

The initial polling runtime:

- re-opens the Host-level peer registry on every cycle;
- refreshes every currently configured peer through the H22 `Refresher`;
- preserves H22 success/failure and credential-redaction behavior;
- isolates peer failures;
- has a conservative fixed polling interval;
- can notify the host observer when material peer state changes.

Re-opening the registry each cycle means a separate `specview federation peer add/remove` process is observed without restarting `specview serve`.

## Multi-host projection

The projection contains:

```text
local Host
configured remote Hosts + freshness
existing H20 federation Projection
```

Rules:

- local snapshot is freshly built for every projection read;
- `fresh`, `stale`, and `unreachable` peers with a cached snapshot contribute that last valid snapshot to H20 aggregation;
- `never_retrieved` peers are visible as Hosts but contribute no repository facts;
- unreachable never means zero sessions or inactive;
- remote freshness metadata does not rewrite source HostSnapshot fields;
- correlation remains the unchanged H20 all-pairs-safe algorithm.

## Initial public surface

H23 adds:

```text
specview federation status
```

which emits a versioned deterministic JSON projection suitable for black-box tests and later MCP/Web reuse.

`specview serve` starts the peer polling runtime alongside the existing local host activity runtime.

## Acceptance criteria

- [ ] polling runtime re-opens peer registry every cycle;
- [ ] peer add/remove can be observed without restarting the runtime;
- [ ] peer failures are isolated from other peers;
- [ ] H22 credential/redaction behavior is reused unchanged;
- [ ] local Host is always represented by a freshly built snapshot;
- [ ] cached remote snapshots remain in projection when a peer is unreachable;
- [ ] never-retrieved peers appear without invented repository facts;
- [ ] source freshness is explicit and deterministic;
- [ ] H20 aggregator is reused without changing correlation semantics;
- [ ] `specview federation status` emits deterministic versioned JSON;
- [ ] language-neutral multi-host projection fixture is consumed in CI;
- [ ] built-binary test covers local + remote fresh + unreachable cached + never-retrieved cases;
- [ ] `specview serve` runs polling with clean cancellation;
- [ ] H18-H22 contracts remain compatible;
- [ ] full gofmt/module/vet/race/coverage/binary/browser/release CI passes.

## Out of scope

- peer discovery;
- push sync;
- remote execution/writes;
- per-peer polling intervals;
- distributed database;
- automatic UI peer management;
- changing HostSnapshot v1;
- changing peer v1/observation v1;
- changing H20 correlation semantics.

## Next

Once H23 proves the runtime/projection boundary, the same read model can be exposed in the host Web UI and MCP without duplicating federation logic.
