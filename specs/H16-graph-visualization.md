---
specview:
  status: in_progress
  depends_on: [H14, H15]
---

# Graph visualization

Visualize the renderer-neutral specification graph in both a practical 2D view and an experimental 3D projection while keeping the graph data model independent from visualization technology.

## 2D graph

The first graph view uses workflow as the horizontal dimension:

```text
New -> In progress -> Done
```

Projects form separate vertical bands in workspace mode.

Nodes remain square and inherit the existing visual language:

- graphite = New.
- amber = In progress.
- green = Done.
- live activity adds an inner square.
- collision adds a red outline.
- idle/orphaned work uses a quiet dashed outline.

Typed dependency edges use arrows. Clicking a node opens the matching specification detail page.

## Experimental 3D graph

The POC uses browser-native canvas and perspective math rather than adding Three.js immediately.

Dimensions are currently:

```text
X = workflow state
Y = row within state
Z = project / repository
```

The user can drag to rotate the scene.

The purpose of the POC is to determine whether project depth and live agent topology provide useful spatial understanding. If the interaction proves valuable, the renderer can later move to Three.js without changing the Markdown relation contract or `/api/graph` API.

## Why not Three.js first

3D is useful only when the dimensions encode real semantics. Building the graph contract and a minimal native renderer first prevents Specview from shipping a visually impressive but semantically empty toy.

A future Three.js renderer may add better camera controls, labels, clustering, large-graph performance, and animation while consuming the same graph JSON.

## Live behavior

The graph should refresh its graph data on visible SSE changes without requiring a full page reload where practical.

Heartbeat-only presence updates must not cause a graph refresh because they do not change visible semantics.

## Acceptance criteria

- `/graph` renders the 2D graph.
- `/graph?mode=3d` renders the experimental 3D projection.
- both consume `/api/graph` rather than reparsing Markdown in JavaScript.
- one graph can represent multiple projects.
- project, workflow, agent activity, orphaned state, and collision state are visible.
- node clicks navigate to the correct project/spec detail.
- unresolved relationships are reported in graph statistics.
- no third-party runtime asset is required for the first 3D experiment.
