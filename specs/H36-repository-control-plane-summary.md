---
specview:
  status: done
---

# H36 - Repository Control Plane Summary

## Goal

Make the repository page the read-only control-plane entrypoint by composing existing Intent, Execution, Evidence, and Acceptance projections into one concise operational snapshot without inventing a new repository health authority.

## Acceptance

- [x] `/project?id=<exact-local-repository-id>` exposes a `repository-control-plane` summary above the existing detailed sections.
- [x] The summary has exactly four first-class facets: Intent, Execution, Evidence, and Acceptance.
- [x] Intent counts come from the existing repository work-item projection and preserve new, in-progress, done, and invalid semantics.
- [x] Execution shows the active logical-session count plus the latest logical session from execution history for the exact local repository.
- [x] The lower `Active now` section remains the existing process-observation view; H36 does not redefine process IDs as logical session identity.
- [x] Evidence uses the H35 native Evidence overview, including passed, failed/error, invalid, and latest-record context.
- [x] Acceptance uses the H33 repository Acceptance overview and preserves configured, accepted, waiting, blocked, unconfigured, and evaluation-pending semantics.
- [x] No synthetic overall repository health, readiness, success, or failure state is introduced.
- [x] Each facet links to its existing authoritative detail surface: Specification, repository History, repository Evidence, or repository Acceptance.
- [x] History, Evidence, and Acceptance links preserve the exact local repository ID.
- [x] Facets degrade independently when their source projection is unavailable; one unavailable facet does not erase the others.
- [x] Native Evidence remains visible even when Intent enrichment is unavailable.
- [x] The summary remains inside the existing `project-live` fragment and therefore refreshes through the existing project SSE material fingerprint.
- [x] The existing project material fingerprint already observes execution sessions, specification state, Acceptance config, and native Evidence; H36 adds no watcher, polling loop, or new network authority.
- [x] Existing Active now, Worktrees, GitHub, and Specification detail sections remain available and keep their current semantics.
- [x] Go tests cover projection composition, latest execution selection, active logical-session count, Evidence and Acceptance counts, and degraded Intent behavior.
- [x] Chromium E2E covers repository control-plane semantics and navigation into the Evidence detail surface.
- [x] No Evidence schema, Acceptance contract, execution-history contract, repository config, Host catalog, SQLite, federation wire model, MCP contract, source-control reader, or write authority changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional CI run #1246 (`32747817212`) passed all gates on commit `cba0bf6a27b8a56d8647790cb5d74942834587d9`:

- total production statement coverage: 65.4%;
- `internal/web` coverage: 50.8%;
- `internal/projectstate` coverage: 73.1%;
- race/unit tests: passed;
- MCP and federation binary smokes: passed;
- Chromium semantic tests: passed;
- release archives and installation command: passed.

The first H36 CI run correctly exposed a process-observation versus logical-session identity mismatch in the Execution facet. The implementation was corrected so the control-plane active count and latest session both come from logical execution history, while the existing `Active now` detail remains the process-observation view.

## Non-goals

- inventing one aggregate repository status or score;
- adding charts or time-series analytics;
- changing existing detail-page semantics;
- adding write actions;
- remote federation aggregation in the local repository summary;
- adding this summary to MCP in this slice.
