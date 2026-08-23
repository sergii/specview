# ADR-006: Model Specview as a graph; treat every UI as a projection

- Status: Accepted
- Date: 2026-08-23

## Context

Specview can render the same underlying software-work state in several useful forms: list, Kanban board, graph, timeline, and later other views. The domain model must not inherit assumptions from any one visualization.

Multi-host development also makes the topology richer than a flat repository list. The same logical repository can exist on a laptop and a devbox at the same time, with different worktrees and concurrent agent sessions. One repository can contain many work items. Git is local source-control state; a remote forge is optional provider context and is singular for one normalized repository identity in the first model.

## Decision

Specview's normalized state is a graph of entities and typed relationships. Views consume projections of this graph.

Conceptual entities:

```text
Host
Repository
RepositoryInstance
Worktree
ExecutionSession
WorkItem
IntentArtifact
GitContext
ForgeContext
Evidence
AcceptanceDecision
```

Core relationships:

```text
Repository
  has_many WorkItem
  has_many RepositoryInstance
  has_one optional ForgeContext

RepositoryInstance
  belongs_to Repository
  belongs_to Host
  has_one GitContext
  has_many Worktree

Worktree
  belongs_to RepositoryInstance
  has_many ExecutionSession

ExecutionSession
  belongs_to RepositoryInstance
  may belong_to Worktree
  may correlate_to WorkItem

WorkItem
  belongs_to Repository
  has_many IntentArtifact
  has_many Evidence
  has_one derived AcceptanceDecision per evaluated revision

Evidence
  belongs_to WorkItem
  belongs_to opaque Revision
```

`RepositoryInstance` means one filesystem realization of a logical Repository on one Host. This distinction is required for cases such as:

```text
sergii/specview
  laptop:/Users/.../specview
  devbox:/home/.../specview
```

Both instances can be active simultaneously.

## Repository identity

Repository identity is not equal to filesystem path and is not equal to GitHub identity.

The first federation design should use a layered identity resolution strategy:

1. explicit Specview identity when configured;
2. normalized repository name as the primary correlation candidate across hosts;
3. Git/forge information as corroborating or disambiguating evidence when available;
4. preserve separate identities when correlation is ambiguous.

A name match alone may generate a candidate relationship but must not silently collapse two unrelated repositories when contradictory Git/forge facts exist.

Repositories without Git or without a remote forge remain first-class entities.

## Git and forge provider

Git and forge context are distinct.

```text
RepositoryInstance -> GitContext
Repository         -> optional ForgeContext
```

The first normalized forge relationship is zero-or-one provider for one repository identity, for example GitHub, GitLab, Bitbucket, or Forgejo. Provider adapters remain mutually exclusive for one normalized remote identity. If future real-world cases require multiple forge mirrors, that is a separate modeled extension rather than an implicit list today.

Git remains valid without any forge provider.

## Work items

A Repository can have zero, one, or many active/durable WorkItems. WorkItem is the normalized concept projected from Specview specs, GitHub Spec Kit changes/specs, OpenSpec changes, or future Intent adapters.

The graph does not require every WorkItem to be visible on a Kanban board. Plane/role semantics from ADR-001 continue to decide which Intent artifacts represent active work versus supporting or durable knowledge.

## Views are replaceable projections

No domain entity is named after a view column, screen, or layout.

Candidate projections include:

```text
List
Kanban
Graph
Timeline
Activity stream
Repository topology
Host topology
Evidence/Acceptance matrix
```

A future Gantt-like or dependency view may exist if the graph contains the necessary temporal/dependency facts. The domain must not invent facts merely to satisfy a visualization.

View-specific ordering, grouping, coordinates, collapsed sections, filters, and styling are presentation state, not canonical domain state.

## Consequences

- the same normalized facts can power web UI, CLI, MCP, future A2A, and multiple visualizations;
- multi-host federation has an explicit place for host-local repository instances;
- repository identity can survive path changes and implementation changes;
- GitHub/Bitbucket/GitLab adapters remain provider projections rather than repository identity itself;
- the Go implementation can later be replaced by Rust without changing graph semantics;
- UI redesigns do not require domain rewrites.

## Migration note

The current `Host -> Repository -> Session` implementation is an intentionally smaller predecessor of this graph. Introducing `RepositoryInstance` and federated logical Repository identity is a future migration. Existing persisted formats must remain readable or receive an explicit versioned migration.

## Non-goals

This ADR does not implement multi-host synchronization, graph storage, a graph database, multiple forge mirrors, dependency scheduling, or a specific graph UI.
