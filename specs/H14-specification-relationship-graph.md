---
specview:
  status: in_progress
  depends_on: [H02, H13]
  blocks: [H16]
---

# Specification relationship graph

Represent explicit relationships between specifications directly in Markdown metadata so workflow topology can be observed without creating a separate project-management database.

## Metadata contract

Specifications may declare:

```yaml
specview:
  status: in_progress
  depends_on:
    - H04
    - H08
  blocks: [H15]
```

The POC accepts list and inline-list forms for `depends_on` and `blocks`.

Relationships are presentation-independent facts. Board, workspace, API, 2D graph, and 3D graph all consume the same parsed relation model.

## Reference resolution

Within a project, relation targets may resolve by:

- stable display ID such as `H04`.
- spec-root-relative path such as `group/H04-example.md`.
- project-relative configured spec path such as `specs/group/H04-example.md`.

Unresolved targets remain observable through the graph API instead of being silently discarded.

## Direction

`A depends_on B` renders an edge from B to A.

`A blocks B` renders an edge from A to B.

Both preserve their relation type in the graph API even when they produce the same visual direction.

## Graph API

`GET /api/graph` returns renderer-neutral JSON containing:

- workspace-wide nodes.
- typed edges.
- project identity.
- spec ID/path/title/status.
- live agent labels.
- derived orphan/collision flags.
- unresolved relation markers.

This contract intentionally separates graph semantics from visualization technology so canvas, SVG, Three.js, or another renderer can evolve independently.

## Acceptance criteria

- relationship metadata remains optional.
- existing status-only specs remain valid.
- duplicate relations are normalized.
- edges resolve by stable ID or supported paths.
- unresolved relationships remain visible in machine-readable graph output.
- graph data works across one or multiple observed projects.
- no graph database is required.
