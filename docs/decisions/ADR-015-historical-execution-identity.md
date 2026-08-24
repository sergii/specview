# ADR-015 - Historical Execution Identity Migration

## Status

Accepted for H26.

## Context

Live Specview Execution already uses logical `ExecutionSession` identity. A session has an adapter, stable session ID, repository/worktree/CWD context, and `process_ids[]`. Process IDs are diagnostics, not identity.

The v0.0.1 Host catalog predates that model. It persists one historical session per `repository + agent + PID`, and the rebuildable SQLite index mirrors that shape. ADR-013 deferred aligning persisted history until after v0.0.1.

## Decision

H26 introduces Host catalog schema v2. Persisted sessions contain:

```text
id
identity_kind       # logical | legacy_pid
adapter
agent
process_ids[]
cwd
worktree_root
started_at
last_seen_at
ended_at
active
```

New product runtime persistence is logical-session based. When Runtime receives a scanner that also implements `ExecutionSource`, it consumes `Sessions()` directly instead of flattening sessions into PID observations. The legacy `Scanner -> []Observation` path remains as an internal compatibility path and writes `legacy_pid` sessions.

## Catalog v1 migration

Catalog v1 remains readable. Each v1 session is migrated in memory by preserving its ID, timestamps, active state, and agent; converting `pid` to `process_ids:[pid]`; inferring known adapters (`Codex -> codex`, `Claude -> claude`, otherwise `legacy`); and setting `identity_kind=legacy_pid`. V1 did not persist CWD/worktree, so those fields remain empty.

Loading v1 marks the catalog material-dirty. The next successful observation or `Flush` writes v2 atomically. Parsing alone need not mutate the file.

## Active migration cutover

On the first live logical observation, active `legacy_pid` fragments may collapse into one logical session only when they belong to the same repository, have compatible agent/adapter identity, and share at least one process ID with the live session.

The replacement keeps the live logical session ID and current live context, while `started_at` becomes the earliest non-zero start among the live observation and matched legacy fragments. Ended legacy history is never guessed or regrouped.

## SQLite index

The SQLite host index is derived and rebuildable. H26 upgrades it to schema v2 with logical-session fields and transactionally rebuilds a v1 index from Host catalog state. It remains non-authoritative.

## Compatibility

H26 does not change Host/repository identity, the live `ExecutionSession` ID algorithm, MCP v1, HostSnapshot v1, federation rules, Web routes, Evidence/Acceptance, or repository config. Catalog v1 remains a supported input contract; catalog v2 becomes canonical persisted output.

## Consequences

Persisted history now aligns with live MCP/Web/federation identity, process churn no longer creates identity churn, and future adapters do not need PID semantics. The cost is a catalog schema increment, a rebuildable SQLite index upgrade, and retained v1-reader compatibility code. Ended v1 history remains explicitly legacy because reconstructing historical logical groups would require guessing.
