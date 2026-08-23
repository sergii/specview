---
specview:
  status: in_progress
---

# H26 - Historical Execution Identity Migration

## Goal

Align persisted Host execution history with the logical `ExecutionSession` model already used by live Web, MCP, and federation projections.

```text
v1 history: repository + agent + PID
                 ↓
v2 history: logical ExecutionSession ID
                 ↓
          process_ids[] diagnostics
```

## Scope

H26 migrates the compatibility Host catalog and rebuildable SQLite host index. It does not change live execution identity, public MCP/federation wire contracts, or product authority boundaries.

## Catalog v2

Canonical persisted session fields:

- `id`
- `identity_kind` (`logical` or `legacy_pid`)
- `adapter`
- `agent`
- `process_ids[]`
- `cwd`
- `worktree_root`
- `started_at`
- `last_seen_at`
- optional `ended_at`
- `active`

Process IDs are sorted, unique positive diagnostics and do not define new logical identity.

## Runtime behavior

- default `ExecutionRegistry` persistence consumes logical `ExecutionSession` objects directly;
- one logical session persists as one historical session regardless of process count;
- process-list changes update diagnostics without changing session ID;
- session lifecycle is keyed by logical session ID;
- legacy Scanner observations remain supported internally and create `legacy_pid` entries only.

## V1 compatibility

- catalog v1 fixture remains readable;
- v1 `pid` migrates to one-element `process_ids`;
- v1 IDs/timestamps/state are preserved;
- adapter is inferred where safe;
- v1 sessions are marked `legacy_pid`;
- loading v1 marks state for v2 persistence but does not require a write merely to parse;
- `Flush` or the next successful observation writes canonical v2.

## Active cutover

- active v1 fragments matching a current live logical session by repository, agent/adapter, and overlapping PID may collapse;
- collapse uses the live logical ID/context;
- earliest known non-zero start time is preserved;
- ended v1 history is never regrouped by guesswork.

## SQLite index

- derived index schema advances to v2;
- v1 index content can be transactionally discarded/rebuilt;
- search semantics remain unchanged;
- index remains non-authoritative.

## Acceptance

- [ ] language-neutral catalog v2 fixture added.
- [ ] catalog v1 fixture remains readable through compatibility tests.
- [ ] v1 load migrates PID to `legacy_pid` + `process_ids` in memory.
- [ ] v1 `Flush` rewrites atomically as catalog v2.
- [ ] one logical live session with multiple processes persists as one catalog session.
- [ ] logical process-list churn preserves session ID.
- [ ] active v1 fragments deterministically collapse into matching logical session.
- [ ] ended v1 fragments remain legacy history.
- [ ] logical session end is keyed by session ID rather than PID.
- [ ] execution view recovers historical `started_at` by logical ID, with legacy-overlap fallback during cutover.
- [ ] SQLite index v1 upgrades/rebuilds to schema v2.
- [ ] SQLite search behavior remains unchanged.
- [ ] H25 heartbeat coalescing semantics remain intact.
- [ ] MCP and HostSnapshot/federation fixtures remain unchanged.
- [ ] race/coverage/build/binary/browser/release CI passes.

## Definition of done

H26 is done when the default product path persists logical sessions end-to-end, catalog v1 upgrades safely to canonical v2, legacy history remains attributable without guessed grouping, the rebuildable index migrates automatically, and all public contracts remain unchanged under full CI.
