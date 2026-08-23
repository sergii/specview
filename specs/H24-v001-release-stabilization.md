---
specview:
  status: in_progress
---

# H24 - v0.0.1 Release Stabilization

## Goal

Freeze feature development after H23 and turn the current Specview control plane into a coherent, installable v0.0.1 release candidate.

H24 adds no new product plane. It exists to reconcile contracts, implementation, documentation, tests, packaging, and installed-user behavior around the product that already exists.

```text
H01-H23 implemented capabilities
              ↓
       contract + debt audit
              ↓
        release CI gates
              ↓
       installable v0.0.1
```

## Product boundary

The canonical product thesis is:

> Specview is a local-first, read-only control plane for observing agentic software development.

The normalized architecture is:

```text
INTENT | EXECUTION | EVIDENCE | ACCEPTANCE
```

with Git/provider context, Host state, MCP, and read-only federation surrounding those planes.

H23 established a derived federation runtime and deterministic local + remote Host projection. Polling changes cached observations; it does not create shared or distributed authority.

Kanban, list, hierarchy, and future graph views are projections over the domain graph. No H24 work may introduce view-specific authority into the domain model.

## Feature freeze

Until v0.0.1 is cut, H24 permits only:

- correctness fixes;
- safety/security fixes;
- portability fixes;
- release/performance fixes;
- contract/documentation reconciliation;
- tests and acceptance gates;
- narrowly scoped refactors that reduce accidental coupling without changing frozen observable behavior.

Explicitly deferred until after v0.0.1:

- new federation transport modes;
- automatic peer discovery;
- push federation sync;
- per-peer polling schedules;
- new agent families;
- new forge providers;
- remote writes or execution;
- new orchestration behavior;
- semantic/vector search;
- a new frontend framework;
- a new view type solely for feature expansion.

## Canonical documentation reconciliation

`SPEC.md` and README describe the architecture implemented through H23.

Acceptance:

- [x] `SPEC.md` describes Intent, Execution, Evidence, and Acceptance.
- [x] `SPEC.md` documents Host observation, Git/provider context, SQLite search, MCP, Host identity, and H20-H22 federation foundations.
- [x] `SPEC.md` documents H23 periodic peer refresh and deterministic multi-host status projection.
- [x] `SPEC.md` states that views are projections over a graph-first domain model.
- [x] `SPEC.md` defines H24 feature freeze and the v0.0.1 release boundary.
- [x] README opening/product architecture matches the canonical specification through H23.
- [x] outdated README/SPEC statements that call H23 federation runtime a future capability are removed.

## Architectural debt audit

H24 explicitly classifies known debt instead of expanding it invisibly.

### D1 - PID-shaped historical catalog sessions

The live Execution contract is logical-session based:

```text
ExecutionAdapter -> ExecutionSession -> ProcessIDs diagnostics
```

The compatibility Host catalog still persists historical sessions around `agent + PID`.

Release decision:

- not a v0.0.1 blocker while live control-plane, MCP, and federation projections remain sourced from normalized ExecutionSession state;
- PID must not be promoted into cross-Host or public logical execution identity;
- migration of historical catalog/index session identity requires an explicit schema/migration slice after v0.0.1 unless installed-product testing exposes a correctness defect.

Acceptance:

- [x] H18 MCP exposes logical session `id` with `process_ids` as separate diagnostics.
- [x] HostSnapshot v1 consumes logical sessions and does not export PID as federation session identity.
- [x] ADR-013 records the post-v0.0.1 historical-session migration target.

### D2 - heartbeat persistence churn

The Host catalog updates `LastSeenAt` during execution scans and currently persists heartbeat-only changes.

H16 Web material fingerprints and H23 federation material fingerprints already suppress heartbeat/transport-only notifications. Disk persistence is a separate compatibility-layer concern.

H24 characterizes the current baseline with `TestCatalogHeartbeatPersistenceBaseline`.

At the current two-second execution scan interval, a continuously active Host can perform up to:

```text
30 catalog snapshots / minute
1,800 catalog snapshots / hour
```

Release decision: **defer write coalescing until after v0.0.1**.

The behavior is bounded to the small atomic JSON compatibility snapshot, does not multiply per browser client, and does not amplify into SQLite/Web/federation material updates. Changing persistence cadence immediately before the first release would alter crash/restart history semantics and deserves a dedicated migration slice.

Acceptance:

- [x] focused baseline test proves heartbeat-only persistence behavior.
- [x] ADR-013 records measured write rate and the `defer` release decision.
- [x] ADR-013 defines the post-v0.0.1 write-coalescing migration boundary.

### D3 - repository-level legacy `server` configuration

`.specview.yaml` still parses `server.host` and `server.port`, while Host observation and federation are conceptually Host-scoped.

Release decision:

- do not silently delete the v1 field before the first release;
- document it as a compatibility artifact;
- future Host settings belong outside repository configuration;
- any removal/migration must be an explicit config-contract version change or backwards-compatible deprecation path.

Acceptance:

- [x] canonical spec classifies `server` as a v1 compatibility artifact.
- [x] README no longer implies repository `server` fields define the future Host configuration model.
- [x] ADR-013 records the post-v0.0.1 config-scope migration boundary.

## Contract audit

H24 verifies that the same facts retain the same semantics across projections.

Required checks:

- [x] Web and MCP share repository/project-state semantics rather than maintaining independent Intent/Evidence/Acceptance models.
- [x] MCP and federation agree that ExecutionSession identity is logical and ProcessIDs are diagnostic.
- [x] Evidence remains revision-scoped across projectstate, Web, MCP, and Acceptance.
- [x] Acceptance remains derived from policy plus Evidence and fails closed when revision identity is unsafe.
- [x] provider checks remain provider context and are not silently treated as normalized Evidence.
- [x] remote HostSnapshot data remains attributable to its source Host and never overrides source/local authority.
- [x] H23 polling/runtime status is derived and never upgrades remote observations into shared authority.
- [x] peer credential values are referenced by environment-variable name and are forbidden from persisted state/error text.

## Release gates

The combined H23 + H24 merge candidate passed:

- [x] shell/script syntax gates;
- [x] gofmt;
- [x] module verification;
- [x] `go vet ./...`;
- [x] `go test -race ./...`;
- [x] production statement coverage gate;
- [x] Go build;
- [x] MCP built-binary stdio smoke;
- [x] federation CLI built-binary smoke;
- [x] federation HTTP built-binary smoke;
- [x] federation peer lifecycle built-binary smoke;
- [x] federation runtime/status built-binary smoke;
- [x] Chromium semantic E2E;
- [x] release archive cross-build for linux/darwin amd64/arm64;
- [x] checksum generation.

Verification baseline before this checklist-only update:

```text
GitHub Actions CI #1014
PR merge candidate: 67efbb9b0bca1be583493b0f48baecd33de17408
branch head: 5868575565c8b31a8d4652f090238642dc01c58e
```

The checklist update itself is documentation-only. Its resulting PR merge candidate must still pass the required CI before H24 is considered ready for installed-user acceptance.

## Artifact preflight

The CI #1014 build artifact for merge candidate `67efbb9` was downloaded and inspected independently from the source checkout.

Acceptance:

- [x] build artifact contains darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 archives and `SHA256SUMS`;
- [x] all generated archive checksums verify;
- [x] every archive contains exactly the expected `specview` binary payload;
- [x] extracted Linux amd64 binary executes `version` and `help`;
- [x] extracted Linux amd64 binary exposes H23 `federation status`;
- [x] extracted Linux amd64 binary passes `serve -> /healthz -> SIGTERM` with clean exit code 0;
- [x] extracted Linux amd64 binary emits valid schema-version-1 local `federation status` JSON from isolated Host state.

These checks are packaging/runtime preflight. They do not replace real macOS installed-user dogfooding.

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
- [ ] H23 `federation status` works from the installed binary;
- [ ] restart preserves intended Host history and peer cache state.

## Release output

When all blocking acceptance criteria are complete:

1. merge H24 to `main`;
2. create the first v0.0.1 tag/release from the green main commit;
3. publish macOS/Linux amd64/arm64 archives and checksums;
4. validate `install.sh` against the published release;
5. dogfood the installed binary before starting H25.

## Definition of done

H24 is done when Specview has one coherent canonical specification through H23, green release gates, an installed-user acceptance result, and every known architectural debt is explicitly either fixed or deferred with a reason.

No H25 feature development begins before that point.
