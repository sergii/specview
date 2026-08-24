---
specview:
  status: in_progress
---

# H36 - Repository Control Plane Summary

## Goal

Make the repository page the read-only control-plane entrypoint by composing existing Intent, Execution, Evidence, and Acceptance projections into one concise operational snapshot without inventing a new repository health authority.

## Acceptance

- [ ] `/project?id=<exact-local-repository-id>` exposes a `repository-control-plane` summary above the existing detailed sections.
- [ ] The summary has exactly four first-class facets: Intent, Execution, Evidence, and Acceptance.
- [ ] Intent counts come from the existing repository work-item projection and preserve new, in-progress, done, and invalid semantics.
- [ ] Execution shows the active logical-session count plus the latest logical session from execution history for the exact local repository.
- [ ] The lower `Active now` section remains the existing process-observation view; H36 does not redefine process IDs as logical session identity.
- [ ] Evidence uses the H35 native Evidence overview, including passed, failed/error, invalid, and latest-record context.
- [ ] Acceptance uses the H33 repository Acceptance overview and preserves configured, accepted, waiting, blocked, unconfigured, and evaluation-pending semantics.
- [ ] No synthetic overall repository health, readiness, success, or failure state is introduced.
- [ ] Each facet links to its existing authoritative detail surface: Specification, repository History, repository Evidence, or repository Acceptance.
- [ ] History, Evidence, and Acceptance links preserve the exact local repository ID.
- [ ] Facets degrade independently when their source projection is unavailable; one unavailable facet does not erase the others.
- [ ] Native Evidence remains visible even when Intent enrichment is unavailable.
- [ ] The summary remains inside the existing `project-live` fragment and therefore refreshes through the existing project SSE material fingerprint.
- [ ] The existing project material fingerprint already observes execution sessions, specification state, Acceptance config, and native Evidence; H36 adds no watcher, polling loop, or new network authority.
- [ ] Existing Active now, Worktrees, GitHub, and Specification detail sections remain available and keep their current semantics.
- [ ] Go tests cover projection composition, latest execution selection, active logical-session count, Evidence and Acceptance counts, and degraded Intent behavior.
- [ ] Chromium E2E covers repository control-plane semantics and navigation into the Evidence detail surface.
- [ ] No Evidence schema, Acceptance contract, execution-history contract, repository config, Host catalog, SQLite, federation wire model, MCP contract, source-control reader, or write authority changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- inventing one aggregate repository status or score;
- adding charts or time-series analytics;
- changing existing detail-page semantics;
- adding write actions;
- remote federation aggregation in the local repository summary;
- adding this summary to MCP in this slice.
