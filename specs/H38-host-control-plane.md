---
specview:
  status: done
---

# H38 - Host Control Plane

## Goal

Compose the repositories already known to one Specview Host into one read-only Web control-plane snapshot across Intent, logical Execution, native Evidence, and Acceptance, while keeping repository-level authorities independent and avoiding any synthetic overall Host health score.

## Acceptance

- [x] The Host page renders one `host-control-plane` projection above the existing repository list/hierarchy.
- [x] The Host control plane exposes exactly four first-class facets: Intent, Execution, Evidence, and Acceptance.
- [x] Intent aggregates normalized WorkItems only from recognized supported repositories and preserves new, in-progress, done, invalid, and unavailable counts.
- [x] Execution aggregates logical execution history, including active logical sessions, active repositories, and the latest logical session.
- [x] H38 does not derive active Execution from the live PID/process observation registry.
- [x] Evidence aggregates native repository Evidence counts across the Host, including passed, failed/error, invalid, affected repositories, and latest-record context.
- [x] Native Evidence remains visible when WorkItem enrichment is unavailable.
- [x] Acceptance aggregates existing repository Acceptance overviews, including configured repositories, accepted, waiting, blocked, unconfigured, invalid, evaluation-pending, and unavailable repository counts.
- [x] Unrecognized ordinary repositories remain visible in the Host repository list but do not become false Acceptance failures or attention items.
- [x] A factual `Needs attention` list links to exact local repositories when projected Intent is invalid/unavailable, Evidence is failed/invalid/unavailable, or Acceptance is blocked/waiting/invalid/unavailable.
- [x] Attention ordering is deterministic by repository recency, then repository name/ID; H38 adds no severity score or synthetic priority.
- [x] The Host-wide control-plane projection remains global when repository search/filtering is active.
- [x] Existing List/Hierarchy switching, repository search, theme, shared Host navigation, and repository drill-down remain unchanged.
- [x] The Execution facet links to the existing `/history` projection; attention rows link to existing local repository pages.
- [x] H38 introduces no write actions, new persistence, watcher, polling loop, network authority, Evidence schema, Acceptance contract, execution-history contract, repository config contract, Host catalog contract, SQLite model, federation wire model, or source-control contract.
- [x] The projection uses the HostServer's injected source-control reader rather than hidden template I/O or a separate authority.
- [x] Unit tests cover healthy composition, failed Evidence plus blocked Acceptance, global search semantics, and an unrecognized repository that must not produce false attention.
- [x] Chromium E2E covers the four Host facets, their deterministic counts, empty attention state, and Host-to-History drill-down.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

- Functional CI #1283 completed successfully on H38 functional head `84fa10de782c4cd176d1fe728dd7dff2eb46897c`.
- `go test -race` passed across `./cmd/... ./internal/...`.
- Total production statement coverage remained 65.5%; `internal/web` increased to 53.1%.
- MCP stdio and all federation binary smokes passed unchanged.
- Chromium semantic tests passed all Host and existing Web flows, including the new Host control-plane projection and Host-to-History drill-down.
- Release archives and installation-command verification passed.
- Earlier CI #1279 exposed only a brittle E2E fixture-name expectation (`sergii/specview` versus the fixture's actual `specview-e2e/repository`); the assertion was made fixture-neutral and production behavior was unchanged.

## Non-goals

- adding Host MCP parity in H38;
- adding a global green/red Host status;
- ranking repositories by a synthesized risk or health score;
- changing repository-level H36/H37 semantics;
- aggregating remote federation Hosts into this local Host projection;
- adding background scans beyond the existing Host refresh lifecycle.
