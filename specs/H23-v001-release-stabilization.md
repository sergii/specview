---
specview:
  status: in_progress
---

# H23 - v0.0.1 Release Stabilization

## Goal

Freeze feature development after H22 and turn the current Specview control plane into a coherent, installable v0.0.1 release candidate.

H23 adds no new product plane. It exists to reconcile contracts, implementation, documentation, tests, and packaging around the product that already exists.

```text
H01-H22 implemented capabilities
              ↓
       contract + debt audit
              ↓
        exact-head CI gates
              ↓
       installable v0.0.1
```

## Product boundary

The canonical product thesis is now:

> Specview is a local-first, read-only control plane for observing agentic software development.

The normalized architecture is:

```text
INTENT | EXECUTION | EVIDENCE | ACCEPTANCE
```

with Git/provider context, Host state, MCP, and read-only federation surrounding those planes.

Kanban, list, hierarchy, and future graph views are projections over the domain graph. No H23 work may introduce view-specific authority into the domain model.

## Feature freeze

Until v0.0.1 is cut, H23 permits only:

- correctness fixes;
- safety/security fixes;
- portability fixes;
- release/performance fixes;
- contract/documentation reconciliation;
- tests and acceptance gates;
- narrowly scoped refactors that reduce accidental coupling without changing frozen observable behavior.

Explicitly deferred until after v0.0.1:

- periodic/background peer refresh;
- new federation transport modes;
- automatic peer discovery;
- new agent families;
- new forge providers;
- remote writes or execution;
- new orchestration behavior;
- semantic/vector search;
- a new frontend framework;
- a new view type solely for feature expansion.

## Canonical documentation reconciliation

`SPEC.md` predates H12-H22 and still describes Specview primarily as a Markdown Kanban POC. H23 updates it to the implemented control-plane architecture.

Acceptance:

- [x] `SPEC.md` describes Intent, Execution, Evidence, and Acceptance.
- [x] `SPEC.md` documents Host observation, Git/provider context, SQLite search, MCP, Host identity, and H20-H22 federation.
- [x] `SPEC.md` states that views are projections over a graph-first domain model.
- [x] `SPEC.md` defines a feature freeze and v0.0.1 release boundary.
- [ ] README opening/product architecture matches the canonical specification.
- [ ] outdated README statements that call implemented capabilities “future” are removed.

## Architectural debt audit

H23 explicitly classifies known debt instead of expanding it invisibly.

### D1 - PID-shaped historical catalog sessions

The live Execution contract is already logical-session based:

```text
ExecutionAdapter -> ExecutionSession -> ProcessIDs diagnostics
```

The compatibility Host catalog still persists historical sessions around `agent + PID`.

Release decision:

- not a v0.0.1 blocker while live control-plane, MCP, and federation projections remain sourced from normalized ExecutionSession state;
- PID must not be promoted into cross-Host or public logical execution identity;
- migration of historical catalog/index session identity requires an explicit schema/migration slice after v0.0.1 unless a correctness defect is found during release testing.

Acceptance:

- [ ] verify no frozen language-neutral MCP/federation contract treats PID as logical session identity;
- [ ] document the post-v0.0.1 migration target.

### D2 - heartbeat persistence churn

The Host catalog updates `LastSeenAt` during execution scans and may persist heartbeat-only changes more frequently than necessary.

Release decision:

- measure first;
- fix before release only if it creates material disk churn, correctness risk, or observable performance problems under normal dogfooding;
- otherwise preserve behavior and schedule write coalescing/throttling after v0.0.1.

Acceptance:

- [ ] add a focused test or instrumentation proving current heartbeat persistence behavior;
- [ ] record release decision as `fix-now` or `defer` with evidence.

### D3 - repository-level legacy `server` configuration

`.specview.yaml` still parses `server.host` and `server.port`, while Host observation and federation are conceptually Host-scoped.

Release decision:

- do not silently delete the v1 field before the first release;
- document it as a compatibility artifact;
- future Host settings belong outside repository configuration;
- any removal/migration must be an explicit config-contract version change or backwards-compatible deprecation path.

Acceptance:

- [x] canonical spec classifies `server` as a v1 compatibility artifact.
- [ ] README no longer implies repository `server` fields define the future Host configuration model.
- [ ] add a post-v0.0.1 config-scope migration note/ADR if not already covered.

## Contract audit

H23 verifies that the same facts retain the same semantics across projections.

Required checks:

- [ ] Web and MCP agree on Repository identity and local Host authority.
- [ ] MCP and federation agree that ExecutionSession identity is logical and ProcessIDs are diagnostic.
- [ ] Evidence remains revision-scoped everywhere.
- [ ] Acceptance remains derived from policy plus Evidence and fails closed when revision identity is unsafe.
- [ ] provider checks are not silently treated as normalized Evidence.
- [ ] remote HostSnapshot data never overrides local/source-Host authority.
- [ ] peer credential values are never persisted or returned in errors.

## Release gates

The exact H23 release head must pass:

- [ ] shell/script syntax gates;
- [ ] gofmt;
- [ ] module verification;
- [ ] `go vet ./...`;
- [ ] `go test -race ./...`;
- [ ] production statement coverage gate;
- [ ] Go build;
- [ ] MCP built-binary stdio smoke;
- [ ] federation CLI built-binary smoke;
- [ ] federation HTTP built-binary smoke;
- [ ] federation peer lifecycle built-binary smoke;
- [ ] Chromium semantic E2E;
- [ ] release archive cross-build for linux/darwin amd64/arm64;
- [ ] checksum generation/verification.

No previous functional head is sufficient. The exact release candidate commit must be green.

## User-install acceptance

The release candidate must be tested as an installed product rather than only inside the source checkout.

Required flow:

```text
publish/use candidate release artifacts
        ↓
install with install.sh
        ↓
run specview outside repo checkout
        ↓
observe real repositories and agents
        ↓
restart and verify Host state
```

Acceptance:

- [ ] install to the default user binary directory succeeds on macOS;
- [ ] installed binary starts Host dashboard without requiring `.specview.yaml` in cwd;
- [ ] Codex/Claude observation works on a real Host where available;
- [ ] repository page shows Git/worktree/provider context without mutating the repository;
- [ ] MCP read-only smoke works against installed binary;
- [ ] one federation peer lifecycle works with secrets supplied only by reference;
- [ ] restart preserves intended Host history and peer cache state.

## Release output

When all blocking acceptance criteria are complete:

1. merge H23 to `main`;
2. create the first v0.0.1 tag/release from the exact green commit;
3. publish macOS/Linux amd64/arm64 archives and checksums;
4. validate `install.sh` against the published release;
5. dogfood the installed binary before starting H24.

## Definition of done

H23 is done when Specview has one coherent canonical specification, exact-head green release gates, an installed-user acceptance result, and every known architectural debt is explicitly either fixed or deferred with a reason.

No H24 feature development begins before that point.
