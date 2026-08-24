---
specview:
  status: in_progress
---

# H29 - Execution History Projection

## Goal

Expose completed and active logical Host execution sessions as one deterministic read-only history projection for Web and MCP without adding persistence or changing execution authority.

## Acceptance

- [ ] A reusable history projection is built from Host catalog v2 only.
- [ ] Each history entry preserves repository attribution and logical session identity.
- [ ] Process IDs remain diagnostics and never become logical identity.
- [ ] Active and ended sessions are both represented.
- [ ] Entries sort by `last_seen_at` descending with deterministic tie-breaking.
- [ ] Ended legacy PID fragments remain separate historical records exactly as H26 migrated them.
- [ ] Web exposes `/history` as a read-only Host execution timeline/list.
- [ ] Web history clearly distinguishes active and ended sessions.
- [ ] Web history links repository-attributed entries back to the local repository page.
- [ ] MCP adds one no-argument read-only tool: `get_execution_history`.
- [ ] MCP returns the same normalized history contract used by Web.
- [ ] Existing MCP tool argument contracts remain unchanged.
- [ ] A language-neutral history fixture freezes projection semantics.
- [ ] Built-binary MCP smoke covers `get_execution_history`.
- [ ] Chromium semantic E2E covers `/history`.
- [ ] No Host catalog, SQLite, federation, Evidence, Acceptance, Git/provider, or repository config wire format changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- execution replay;
- agent orchestration;
- remote execution history federation;
- deleting or pruning history;
- new history persistence;
- duration analytics or billing;
- automatic semantic grouping of legacy PID fragments.
