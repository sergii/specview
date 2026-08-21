# ADR-001: Model software work as Intent, Execution, and Evidence

- Status: Accepted
- Date: 2026-08-21

## Context

Specview started as a read-only filesystem observer for Markdown specifications. That vertical slice is intentionally small and remains valid for v0.0.1.

The longer-term product needs to observe more than a specification status. Repo-native, spec-driven AI development has three distinct kinds of information:

1. what the software is supposed to become;
2. what is happening right now while it is being changed;
3. why the resulting change should be considered acceptable.

Mixing those concerns into one status field or one UI layout would couple the domain model to the current dashboard and make integrations with different spec-driven-development conventions difficult.

## Decision

Specview models every unit of software work through three orthogonal dimensions:

```text
INTENT                    EXECUTION                 EVIDENCE
what should be true?      what is happening now?   why should we trust it?
```

### Intent

Intent is durable, declared information, primarily derived from repository artifacts.

Examples:

- specification;
- requirements;
- acceptance criteria;
- examples and scenarios;
- proposal;
- RFC;
- ADR or another decision record;
- design;
- plan;
- tasks;
- project constitution, steering rules, or other policy;
- dependencies and constraints.

Intent is not limited to a single directory layout or Markdown convention.

### Execution

Execution is ephemeral observed state.

Examples:

- an agent or human has claimed work;
- current task;
- active workspace or Git worktree;
- branch;
- process or container state;
- last activity;
- pull request state;
- runtime transitions.

Execution should be reconstructed from adapters and runtime facts rather than manually copied into durable specification files whenever possible.

### Evidence

Evidence is proof produced by independent verification mechanisms.

Examples:

- compiler or type checker result;
- linter result;
- unit, integration, acceptance, contract, property, or fuzz tests;
- mutation testing;
- architecture-boundary checks;
- security analysis;
- migration safety checks;
- performance or load tests;
- hardware-in-the-loop verification;
- AI or human review;
- CI result.

Evidence is project-specific. A landing page and an industrial IoT system may use the same Specview core with very different acceptance policies.

## Workflow is a separate projection

Intent, Execution, and Evidence are not workflow columns.

A UI may project a work item into a lifecycle such as:

```text
TODO -> IN PROGRESS -> ACCEPTANCE -> IN REVIEW -> DONE
```

Conditions such as `blocked`, `waiting`, `failed`, `stale`, or `drift` are orthogonal signals, not additional mandatory workflow columns.

The UI is a projection of the normalized model. List, board, graph, timeline, evidence, and workspace views may change without changing the underlying contracts.

## Authority by fact type

Specview does not define one global source of truth for every fact.

- Filesystem and Git are authoritative for repo-native intent artifacts and their history.
- GitHub, GitLab, or another SCM adapter is authoritative for remote PR, review, merge, and remote CI facts that it owns.
- Runtime adapters are authoritative for observed process, agent, workspace, container, and service facts.
- SQLite is a rebuildable projection, cache, search index, and correlation store. It is not the canonical source of intent.

Deleting the local Specview database must not destroy canonical project knowledge.

## Adapter boundary

Specview normalizes external conventions instead of forcing every project into one directory format.

The native Specview convention is owned by `SpecviewAdapter`:

```text
specs/
  H01.md
  H02.md
```

`SpecviewAdapter` is the zero-config/default adapter and preserves the original v0.0.1 behavior. It is not a generic name for every Markdown layout.

Initial adapter classes:

- artifact adapters: SpecviewAdapter, GitHub Spec Kit, OpenSpec, Kiro, BMAD, and future custom company adapters;
- SCM adapters: Git, GitHub, later GitLab and others;
- execution adapters: filesystem, generic processes, Codex, Claude Code, OpenCode, Docker, systemd, and others;
- verification adapters: test, lint, security, architecture, performance, hardware, and other evidence producers;
- review adapters: human, AI reviewer, GitHub review, and other approval mechanisms.

A future custom Markdown adapter may observe arbitrary company-specific directories, but it is semantically distinct from `SpecviewAdapter`.

Adapters emit normalized facts/events. The core domain model must not depend on adapter-specific file paths or tool names.

## Consequences

- The existing v0.0.1 filesystem watcher becomes the implementation behind `SpecviewAdapter`.
- The current dashboard can remain unchanged while the domain model evolves underneath it.
- Future SQLite support can index and correlate artifacts without becoming their canonical store.
- A future Go-to-Rust rewrite can preserve the same semantic and adapter contracts.
- Specview can support multiple spec-driven-development frameworks without becoming a new competing spec format.
- Acceptance becomes evidence-driven and policy-driven rather than hard-coded to a particular test stack.

## Non-goals

This ADR does not add Jira-like project-management features, specification editing, an embedded LLM, a scheduler, or automatic workflow mutation to v0.0.1.
