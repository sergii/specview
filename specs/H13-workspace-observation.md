---
specview:
  status: in_progress
  depends_on: [H08, H10, H12]
  blocks: [H14]
---

# Multi-project workspace observation

Observe more than one configured project in a single Specview process without introducing a database or changing the per-project Markdown source-of-truth model.

## Project discovery

`specview` and `specview serve` continue to observe the current configured project.

Workspace mode accepts one or more paths:

```bash
specview serve ~/repos/sergii ~/repos/spotwo
```

Discovery is deliberately bounded for the POC:

- a supplied directory containing `.specview.yaml` is one project.
- otherwise, Specview checks only immediate child directories for `.specview.yaml`.
- hidden directories are skipped during collection discovery.
- no recursive crawl through dependency/vendor/build trees.
- explicit project paths can be mixed with collection folders.

Each discovered project keeps its own configuration, specification store, activity store, specification watcher, and activity watcher. All projects share the same HTTP server and SSE hub.

## Project identity

The canonical project identity remains the full resolved filesystem path.

The board presents only the final two path components, for example:

```text
sergii/specview
spotwo/research
uisorg/uis-support
```

This normally provides enough namespace context to distinguish personal, company, and similarly named repositories without filling the header with machine-specific prefixes.

The full path remains available as hover/title metadata and must remain available internally for correlation.

## Workspace presentation

The existing three-column project board is a reusable `ProjectView`.

A workspace renders:

```text
Workspace summary

ProjectView A
ProjectView B
ProjectView C
...
```

For multiple projects, the summary shows compact project identity plus useful current counts:

- New.
- In progress.
- Done.
- active agent sessions.
- idle/orphaned in-progress specifications.
- collision-affected specifications.

The same Classic / Dense / Flow presentation preference applies to every project block.

## Routing

Specification links include a deterministic project key as well as the project-relative spec path so identically named specs in different projects do not collide:

```text
/spec?project=p-...&path=H12-agent-activity-observation.md
```

## Acceptance criteria

- one-project behavior remains compatible with the existing dashboard.
- multiple configured projects can be observed by one Specview process.
- workspace discovery is bounded and predictable.
- project display identity is `parent/current` while the full resolved path remains available on hover.
- each project retains independent spec/activity stores and watchers.
- project detail routing remains unambiguous across duplicate spec paths.
- workspace summary exposes agent-first operational signals, not only workflow counts.
- no database or central write model is introduced.
