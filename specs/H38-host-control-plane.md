---
specview:
  status: in_progress
---

# H38 - Host Control Plane

## Goal

Compose the repositories already known to one Specview Host into one read-only Web control-plane snapshot across Intent, logical Execution, native Evidence, and Acceptance, while keeping repository-level authorities independent and avoiding any synthetic overall Host health score.

## Acceptance

- [ ] The Host page renders one `host-control-plane` projection above the existing repository list/hierarchy.
- [ ] The Host control plane exposes exactly four first-class facets: Intent, Execution, Evidence, and Acceptance.
- [ ] Intent aggregates normalized WorkItems only from recognized supported repositories and preserves new, in-progress, done, invalid, and unavailable counts.
- [ ] Execution aggregates logical execution history, including active logical sessions, active repositories, and the latest logical session.
- [ ] H38 does not derive active Execution from the live PID/process observation registry.
- [ ] Evidence aggregates native repository Evidence counts across the Host, including passed, failed/error, invalid, affected repositories, and latest-record context.
- [ ] Native Evidence remains visible when WorkItem enrichment is unavailable.
- [ ] Acceptance aggregates existing repository Acceptance overviews, including configured repositories, accepted, waiting, blocked, unconfigured, invalid, evaluation-pending, and unavailable repository counts.
- [ ] Unrecognized ordinary repositories remain visible in the Host repository list but do not become false Acceptance failures or attention items.
- [ ] A factual `Needs attention` list links to exact local repositories when projected Intent is invalid/unavailable, Evidence is failed/invalid/unavailable, or Acceptance is blocked/waiting/invalid/unavailable.
- [ ] Attention ordering is deterministic by repository recency, then repository name/ID; H38 adds no severity score or synthetic priority.
- [ ] The Host-wide control-plane projection remains global when repository search/filtering is active.
- [ ] Existing List/Hierarchy switching, repository search, theme, shared Host navigation, and repository drill-down remain unchanged.
- [ ] The Execution facet links to the existing `/history` projection; attention rows link to existing local repository pages.
- [ ] H38 introduces no write actions, new persistence, watcher, polling loop, network authority, Evidence schema, Acceptance contract, execution-history contract, repository config contract, Host catalog contract, SQLite model, federation wire model, or source-control contract.
- [ ] The projection uses the HostServer's injected source-control reader rather than hidden template I/O or a separate authority.
- [ ] Unit tests cover healthy composition, failed Evidence plus blocked Acceptance, and an unrecognized repository that must not produce false attention.
- [ ] Chromium E2E covers the four Host facets, their deterministic counts, empty attention state, and Host-to-History drill-down.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- adding Host MCP parity in H38;
- adding a global green/red Host status;
- ranking repositories by a synthesized risk or health score;
- changing repository-level H36/H37 semantics;
- aggregating remote federation Hosts into this local Host projection;
- adding background scans beyond the existing Host refresh lifecycle.
