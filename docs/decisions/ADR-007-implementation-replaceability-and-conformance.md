# ADR-007: Keep Specview implementation-replaceable and prove conformance with language-neutral contracts

- Status: Accepted
- Date: 2026-08-23

## Context

Specview is currently implemented in Go. A future iteration is expected to move substantial or complete implementation to Rust.

A rewrite is safe only if the observable contracts and domain semantics are more stable than the implementation language. Go unit tests alone are insufficient because a Rust implementation cannot reuse Go package APIs as its compatibility oracle.

## Decision

Specview treats implementation language as replaceable infrastructure behind versioned semantic and wire contracts.

The following layers are ordered from most stable to least stable:

```text
1. domain semantics and authority rules
2. versioned wire/persistence contracts
3. public CLI / HTTP / MCP behavior
4. adapter contracts
5. implementation package structure
6. UI layout and styling
```

A Go-to-Rust migration must preserve layers 1-4 unless a deliberate versioned contract change is accepted.

## Normative conformance fixtures

Repository-owned fixtures under `testdata/contracts/` are language-neutral compatibility examples. They are not Go-specific test data.

Initial frozen contracts are only formats that already exist today:

```text
.specview.yaml version 1
native Evidence JSON version 1
host catalog JSON version 1
```

The Go implementation must parse these fixtures in CI. A future Rust implementation must run the same fixtures before replacing Go behavior.

We deliberately do not freeze a future graph/MCP JSON schema before its model is designed. When a public JSON/MCP representation is introduced, versioned fixtures and conformance tests must be added at the same time.

## Compatibility rules

- Existing versioned input must not change meaning silently.
- Unknown fields are rejected where the current contract is strict.
- Persistence format changes require a version bump or explicit migration path.
- Stable normalized enum values must not be renamed without a contract version change.
- Provider-specific details must not become core semantic requirements.
- Filesystem paths and process IDs are runtime facts, not cross-host logical identity.
- A rewrite may change internal algorithms, storage engines, concurrency models, and package/module boundaries while preserving observable semantics.

## Test pyramid for rewrite safety

### Tier 1 - domain/unit tests

Fast tests for normalization, evaluation, identity rules, adapters, and fail-closed behavior.

### Tier 2 - language-neutral contract fixtures

Golden YAML/JSON inputs and expected normalized meaning. These are the primary Go-to-Rust conformance safety net.

### Tier 3 - black-box integration tests

Exercise process boundaries and public behavior such as CLI, persisted state, HTTP fragments, SSE semantics, and later MCP resources/tools.

### Tier 4 - browser semantic E2E

Use Playwright when the UI has multiple mature views. Assert user-observable behavior, navigation, live updates, accessibility roles, and stable data relationships rather than implementation DOM structure.

### Tier 5 - visual regression snapshots

Use Playwright screenshot comparisons for a small set of canonical stable screens and states. Visual snapshots are presentation regression tests, not domain contracts.

They should be introduced after the relevant view is stable enough that intentional design movement does not create constant noise. Dynamic values such as ages, timestamps, hostnames, process IDs, and blinking/live indicators should be fixed or masked.

## Visual snapshots are not rewrite gates by themselves

A Rust rewrite may preserve semantics while intentionally changing HTML generation or CSS. Therefore pixel snapshots cannot be the only acceptance criterion for implementation parity.

For mature views, the preferred order is:

```text
semantic E2E passes
contract fixtures pass
then visual snapshots confirm intended appearance
```

## CI direction

The normal CI gate continues to run formatting, module hygiene, vet, race tests, and build for the Go implementation. Contract conformance tests are part of the normal `go test ./...` suite so they cannot be skipped accidentally.

When Rust is introduced, CI should temporarily run Go and Rust implementations against the same contract fixtures and black-box scenarios. Cutover occurs only after parity gates pass.

## Consequences

- Rust can replace Go incrementally rather than through a risky flag day;
- existing users keep compatible config/evidence/persistence semantics;
- adapters and protocols depend on normalized contracts rather than Go types;
- test fixtures become executable product documentation;
- visual UI tests remain useful without coupling the product core to pixels.

## Non-goals

This ADR does not choose the Rust web framework, async runtime, SQLite crate, MCP SDK, migration date, or whether Go and Rust binaries coexist during the transition.
