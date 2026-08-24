---
specview:
  status: in_progress
---

# H37 - Repository Control Plane MCP Parity

## Goal

Expose the H36 repository control-plane snapshot to MCP clients as one additive read-only tool over the existing Intent, logical Execution, native Evidence, and Acceptance authorities without changing existing MCP payloads or inventing aggregate repository health.

## Acceptance

- [ ] MCP advertises `get_repository_control_plane` when the configured Reader implements repository control-plane reads.
- [ ] `get_repository_control_plane` accepts exactly one `repository_id` argument using the existing opaque host-local repository ID contract.
- [ ] The structured result contains exactly four first-class facets: `intent`, `execution`, `evidence`, and `acceptance` plus repository identity metadata.
- [ ] Intent counts preserve new, in-progress, done, and invalid semantics from the existing WorkItem projection.
- [ ] Execution counts active logical sessions and exposes the latest logical session from execution history for the exact local repository.
- [ ] H37 does not derive logical active-session counts from the live PID/process observation registry.
- [ ] Evidence uses the native repository Evidence overview, preserves failed/error and invalid counts, and exposes latest-record context.
- [ ] Native Evidence remains available when WorkItem enrichment is unavailable.
- [ ] Acceptance uses the repository Acceptance overview and preserves configured, revision, evidence-count, accepted, waiting, blocked, unconfigured, invalid, and evaluation-pending semantics.
- [ ] Facets degrade independently through facet-local error fields where their source projection is unavailable.
- [ ] No synthetic overall repository health, readiness, success, or failure state is introduced.
- [ ] Existing MCP tools and their argument contracts are unchanged.
- [ ] MCP v4 is strictly additive over the v3 tool contract by exactly one tool.
- [ ] Readers that do not implement repository control-plane reads do not advertise the H37 tool, preserving existing test/client capability behavior.
- [ ] Unit tests cover repository projection composition, active logical-session semantics, latest session selection, Evidence and Acceptance counts, and degraded Intent with surviving native Evidence.
- [ ] MCP server tests cover capability advertisement and structured H37 tool output.
- [ ] Production MCP binary smoke invokes `get_repository_control_plane` and validates the structured projection.
- [ ] No Evidence schema, Acceptance contract, execution-history contract, repository config, Host catalog, SQLite, federation wire model, source-control contract, watcher, polling loop, network authority, or write authority changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- changing the H36 Web layout;
- replacing `get_repository`;
- adding write actions;
- adding remote federation aggregation to the local repository control-plane tool;
- inventing one aggregate repository status or score;
- changing MCP protocol version `2025-11-25`.
