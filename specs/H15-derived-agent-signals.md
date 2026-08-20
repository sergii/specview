---
specview:
  status: in_progress
  depends_on: [H12, H13]
  blocks: [H16]
---

# Derived agent-first signals

Derive operational signals from workflow metadata and ephemeral agent presence without creating new persisted workflow statuses.

## Idle / orphaned work

A specification is presented as `idle` when all of the following are true:

- workflow status is `in_progress`.
- no fresh live agent session is attached.
- the specification file has not been modified for at least 15 minutes.

This is a heuristic observation, not a canonical state. It must never rewrite Markdown automatically.

## Agent collision

If more than one fresh activity session is attached to the same specification, Specview surfaces an `agent collision` signal.

Concurrent agents are not automatically an error, but the condition is important enough to make visible.

## Code collision

Activity records may optionally declare the project-relative files currently touched by the agent:

```json
{
  "files": [
    "internal/web/server.go",
    "internal/activity/activity.go"
  ]
}
```

If live sessions on different specifications within one project report at least one identical file, both specifications surface `overlap <n>` with the overlapping files available as hover metadata.

No file list means collision is unknown, not that the work is collision-free.

## Workspace aggregation

Workspace summary counts:

- fresh active sessions.
- idle/orphaned in-progress specs.
- specs affected by either agent or code collision.

A specification affected by both collision types is counted once in the project collision total.

## Acceptance criteria

- signals are derived and read-only.
- workflow status remains New / In progress / Done.
- heartbeat-only changes do not create browser reloads.
- file-set changes do trigger a visible refresh because collision semantics may change.
- idle/orphaned is clearly described as a heuristic.
- code collision requires explicit file evidence.
- collision signals are available in board/workspace and graph projections.
