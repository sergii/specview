---
specview:
  status: done
---

# H23 - Federation Runtime and Multi-Host Projection

## Goal

Keep configured federation peers current while `specview serve` is running and expose one deterministic read-only multi-host projection without changing H20-H22 contracts.

## Runtime

The polling runtime:

- re-opens the Host-level peer registry on every cycle;
- refreshes every currently configured peer through the H22 `Refresher`;
- preserves H22 success/failure and credential-redaction behavior;
- isolates peer failures;
- uses a conservative fixed polling interval;
- notifies the host observer only when peer material changes.

Re-opening the registry each cycle means a separate `specview federation peer add/remove` process is observed without restarting `specview serve`.

Material-change notification excludes transport-only attempt timestamps, preventing periodic polling from causing unnecessary UI reloads when peer facts are unchanged.

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

## Public surface

H23 adds:

```text
specview federation status
```

which emits a versioned deterministic JSON projection suitable for black-box tests and later MCP/Web reuse.

`specview serve` starts the peer polling runtime alongside the existing local host activity runtime.

## Acceptance criteria

- [x] polling runtime re-opens peer registry every cycle;
- [x] peer add/remove can be observed without restarting the runtime;
- [x] peer failures are isolated from other peers;
- [x] H22 credential/redaction behavior is reused unchanged;
- [x] local Host is always represented by a freshly built snapshot;
- [x] cached remote snapshots remain in projection when a peer is unreachable;
- [x] never-retrieved peers appear without invented repository facts;
- [x] source freshness is explicit and deterministic;
- [x] H20 aggregator is reused without changing correlation semantics;
- [x] `specview federation status` emits deterministic versioned JSON;
- [x] language-neutral multi-host projection fixture is consumed in CI;
- [x] built-binary test covers local + remote fresh + unreachable cached + never-retrieved cases;
- [x] `specview serve` performs an initial peer poll and shuts polling down cleanly;
- [x] host observer notifications occur only on material federation changes;
- [x] H18-H22 contracts remain compatible;
- [x] full gofmt/module/vet/race/coverage/binary/browser/release CI passes.

## Verification baseline

Functional head `85a469f314abd781656eca0522991bf48d7dd38e` passed CI #993 with:

- gofmt and module hygiene;
- go vet;
- race tests;
- 63.4% total production statement coverage;
- 80.0% `internal/federationruntime` coverage;
- build;
- MCP binary stdio smoke;
- federation file/CLI binary smoke;
- federation HTTP binary smoke;
- federation peer lifecycle binary smoke;
- federation runtime/status binary smoke, including fresh, unreachable-cached, never-retrieved, initial polling, and clean process shutdown;
- Chromium semantic E2E;
- release archives.

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

The same multi-host read model can now be exposed in the host Web UI and MCP without duplicating federation logic.
