---
specview:
  status: done
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

- active v1 fragments matching a current live logical session by repository filesystem root, agent/adapter, and overlapping PID may collapse;
- a persisted repository record with the same filesystem root is reused even when its legacy ID differs from the current derived root ID;
- multiple persisted repository records for the same filesystem root fail closed rather than attaching a live session nondeterministically;
- collapse uses the live logical session ID/context;
- earliest known non-zero start time is preserved;
- ended v1 history is never regrouped by guesswork.

## SQLite index

- derived index schema advances to v2;
- v1 index content is transactionally discarded and rebuilt from authoritative catalog state;
- search semantics remain unchanged;
- index remains non-authoritative.

## Acceptance

- [x] language-neutral catalog v2 fixture added.
- [x] catalog v1 fixture remains readable through compatibility tests.
- [x] v1 load migrates PID to `legacy_pid` + `process_ids` in memory.
- [x] v1 `Flush` rewrites atomically as catalog v2.
- [x] one logical live session with multiple processes persists as one catalog session.
- [x] logical process-list churn preserves session ID.
- [x] default Runtime prefers logical `ExecutionSource.Sessions()` and does not flatten through legacy `Scan()`.
- [x] active v1 fragments deterministically collapse into matching logical session.
- [x] v1-to-live cutover preserves one repository record per filesystem root and ended legacy history.
- [x] ended v1 fragments remain legacy history.
- [x] logical session end is keyed by session ID rather than PID.
- [x] execution view recovers historical `started_at` by logical ID, with legacy-overlap fallback during cutover.
- [x] SQLite index v1 upgrades/rebuilds to schema v2.
- [x] SQLite search behavior remains unchanged.
- [x] H25 heartbeat coalescing semantics remain intact.
- [x] MCP and HostSnapshot/federation fixtures remain unchanged.
- [x] race/coverage/build/binary/browser/release CI passes.

## Verification

Functional head `e9d0c07135e0aa696488ed50af00ec6cc7e4c530` passed CI run #1084 (`32675578256`) with the complete matrix:

- formatting/modules/vet/race: pass
- total production statement coverage: **65.0%**
- `internal/hoststate`: **65.6%**
- `internal/hostindex`: **71.3%**
- build: pass
- MCP built-binary stdio smoke: pass
- federation CLI/HTTP/peer/runtime built-binary smokes: pass
- Chromium semantic E2E: pass
- release archive cross-build: pass

Diff audit from H25 merge `30a908304c5a68b49f67fc98adcf31212c2d0373` changes only Host catalog/runtime execution persistence, Web material fingerprinting, the rebuildable SQLite projection, tests, fixtures, and H26 documentation. `controlplane`, MCP server, federation packages, and HostSnapshot wire contracts are unchanged.

## Definition of done

H26 is done: the default product path persists logical sessions end-to-end, catalog v1 upgrades safely to canonical v2, legacy history remains attributable without guessed grouping, the rebuildable index migrates automatically, and all public contracts remain unchanged under full CI.
