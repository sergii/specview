---
specview:
  status: done
---

# H29 - Execution History Projection

## Goal

Expose completed and active logical Host execution sessions as one deterministic read-only history projection for Web and MCP without adding persistence or changing execution authority.

## Acceptance

- [x] A reusable history projection is built from Host catalog v2 only.
- [x] Each history entry preserves repository attribution and logical session identity.
- [x] Process IDs remain diagnostics and never become logical identity.
- [x] Active and ended sessions are both represented.
- [x] Entries sort by `last_seen_at` descending with deterministic tie-breaking.
- [x] Ended legacy PID fragments remain separate historical records exactly as H26 migrated them.
- [x] Web exposes `/history` as a read-only Host execution timeline/list.
- [x] Web history clearly distinguishes active and ended sessions.
- [x] Web history links repository-attributed entries back to the local repository page.
- [x] MCP adds one no-argument read-only tool: `get_execution_history`.
- [x] MCP returns the same normalized history contract used by Web.
- [x] Existing MCP tool argument contracts remain unchanged.
- [x] A language-neutral history fixture freezes projection semantics.
- [x] Built-binary MCP smoke covers `get_execution_history`.
- [x] Chromium semantic E2E covers `/history`.
- [x] No Host catalog, SQLite, federation, Evidence, Acceptance, Git/provider, or repository config wire format changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional head `fa735373f81db00bbccb7fd3d8bb8bc9aa706dfc` passed CI #1169 (`32710587999`) completely:

- formatting/modules/vet/race ✅
- total production coverage: **64.8%** ✅
- `internal/executionhistory`: **76.5%**
- `internal/mcpserver`: **76.2%**
- `internal/web`: **41.7%**
- build ✅
- built-binary MCP stdio smoke including `get_execution_history` ✅
- federation CLI/HTTP/peer/runtime binary smokes ✅
- Chromium semantic E2E for `/history` ✅
- Linux/macOS amd64/arm64 release archive cross-build ✅

## Non-goals

- execution replay;
- agent orchestration;
- remote execution history federation;
- deleting or pruning history;
- new history persistence;
- duration analytics or billing;
- automatic semantic grouping of legacy PID fragments.
