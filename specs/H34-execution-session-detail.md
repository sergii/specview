---
specview:
  status: done
---

# H34 - Execution Session Detail

## Goal

Make each active or ended logical execution session directly inspectable from execution history while preserving the Host catalog and execution-history projection as the only execution authority.

## Acceptance

- [x] Every execution-history row links to `/history/session?repository=<exact-repository-id>&session=<exact-session-id>`.
- [x] Session lookup requires the exact repository and session pair and cannot match the same session ID in another repository.
- [x] Unknown or incomplete session locators return 404 instead of falling back to another session.
- [x] The detail renders existing logical identity, adapter, agent, repository root, worktree root, CWD, sorted process diagnostics, lifecycle timestamps, active/ended state, and observed duration.
- [x] Process IDs remain explicitly diagnostic and are not promoted into execution identity.
- [x] Active and ended sessions use the same read-only detail surface and existing history semantics.
- [x] Session detail keeps History as the active top-level Host view.
- [x] Session detail preserves exact repository context in History and Acceptance navigation and links directly to repository, repository history, and global history.
- [x] The detail is derived only from the existing `executionhistory` projection and does not inspect live processes or introduce another execution reader.
- [x] Go tests cover exact repository+session lookup, rendered detail semantics, and wrong-pair rejection.
- [x] Chromium E2E covers History -> active session detail -> repository-scoped History and verifies logical identity plus process diagnostics.
- [x] No Host catalog schema, execution-history JSON schema, MCP contract, federation contract, SQLite, repository config, Evidence, Acceptance, or write-authority changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional CI run #1206 passed all jobs. Production statement coverage was 65.1% total, with `internal/executionhistory` at 70.0% and `internal/web` at 48.1%. Chromium semantic tests and release archive cross-builds passed.

## Non-goals

- process control, termination, or signals;
- live process introspection outside the existing Host catalog;
- session logs or transcript storage;
- remote federation session detail;
- changing logical-session identity rules;
- adding a new MCP method in this slice.
