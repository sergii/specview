---
specview:
  status: new
---

# Normalized artifact adapters

Introduce the first architecture boundary that separates repository-specific spec layouts from the Specview domain model.

## Intent

Specview must be able to observe repo-native specification systems without requiring every project to store files in the same directory structure.

The current `specs/*.md` behavior remains the first Generic Markdown adapter and must continue to work.

The core vocabulary is:

```text
INTENT | EXECUTION | EVIDENCE
```

This slice is limited to the Intent side of that model.

## Scope

Define a normalized artifact model and adapter boundary for repository artifacts.

Initial normalized concepts:

- artifact;
- artifact kind;
- work item/capability/change identity when derivable;
- source path;
- declared metadata;
- relation between artifacts.

Initial artifact kinds should be able to represent:

- policy;
- proposal;
- spec;
- requirement;
- example;
- RFC;
- decision/ADR;
- design;
- plan;
- task;
- contract;
- research;
- checklist.

Not every kind needs a dedicated UI in this slice.

## Adapter behavior

The first implementation must support:

1. Generic Markdown using the existing configured `specs.path`;
2. explicit adapter selection in configuration;
3. zero-config convention detection where unambiguous;
4. normalized artifacts that contain no framework-specific path assumptions in the UI layer;
5. normalized relations such as `depends_on`, `relates_to`, `resolves`, and `supersedes` when supplied by an adapter;
6. graceful fallback to Generic Markdown when no supported framework convention is detected.

Candidate later adapters:

- GitHub Spec Kit;
- OpenSpec;
- Kiro;
- BMAD;
- custom company layouts.

## Current truth vs work in flight

The normalized model must be capable of representing the OpenSpec-style distinction between:

```text
CURRENT KNOWLEDGE
```

and:

```text
CHANGE IN FLIGHT
```

The first implementation does not need to render a dedicated UI for this distinction, but it must not choose a schema that makes the distinction impossible later.

## SQLite

A future SQLite database may index normalized artifacts, relations, full-text search, and derived state.

SQLite is explicitly not the canonical source of repository intent. The projection must be rebuildable from authoritative adapters.

SQLite implementation is outside this slice unless needed for a minimal proof of the adapter boundary.

## UI

Do not redesign the dashboard as part of this slice.

The current UI is a projection. The adapter/core boundary should allow future list, board, graph, timeline, evidence, and workspace views without changing input contracts.

## Acceptance

- existing Generic Markdown `specs/*.md` projects continue to render unchanged;
- artifact discovery is behind an adapter interface rather than embedded in the UI;
- normalized artifacts expose stable semantic kinds independent of directory names;
- normalized relations can be represented even if the current UI does not render all of them;
- explicit configuration overrides auto-detection;
- unsupported layouts remain observable through the Generic Markdown fallback when configured;
- no Jira/Linear task-management behavior is introduced;
- no specification editing is introduced;
- no embedded LLM is introduced;
- documentation references ADR-001 as the governing semantic model.

## References

- `docs/decisions/ADR-001-intent-execution-evidence.md`
- `docs/research/spec-driven-layouts.md`
