# ADR-005: Use MCP first, A2A later, and treat Agent Client Protocol separately

- Status: Accepted
- Date: 2026-08-23

## Context

Specview is evolving from a specification dashboard into a local-first execution control plane for agentic software development.

A developer may have many repositories and concurrent coding-agent sessions spread across a workstation, devbox, and later other hosts. Coding agents need a standard way to ask Specview factual questions such as:

```text
Which repositories are active?
Which worktrees exist?
Which agents are running and where?
What changed today?
Which work items are blocked?
What evidence exists for this revision?
```

Specview should expose these facts without embedding a model and without coupling the core domain to one agent vendor.

The acronym `ACP` is currently overloaded:

1. the IBM/BeeAI Agent Communication Protocol for agent-to-agent interoperability has moved into the A2A ecosystem;
2. Agent Client Protocol is a separate protocol, developed for editor/IDE-to-coding-agent communication.

Those solve different problems and must not be treated as one integration target.

## Decision

Specview adopts a layered protocol strategy.

```text
Core domain
  Intent
  Execution
  Evidence
  Acceptance
  Host/Repository state
        │
        ├── local UI / CLI
        ├── MCP
        ├── future A2A
        └── optional Agent Client Protocol adapter
```

Protocol adapters consume the normalized core. Protocol-specific concepts must not leak into the core domain model.

## MCP is first

The first agent-facing protocol is Model Context Protocol.

MCP matches Specview's immediate use case: an existing coding agent needs structured read-only access to deterministic Specview facts.

The first MCP server should expose read operations only.

Candidate tools/resources:

```text
list_hosts
get_host
list_repositories
get_repository
search_repositories
list_active_sessions
get_session
list_worktrees
get_work_item
get_evidence
get_acceptance
```

The initial implementation should run locally and reuse the same in-memory/catalog/index services as the web UI rather than constructing a parallel state model.

Specview remains useful without MCP. MCP is an adapter, not the product core.

## No embedded LLM

The MCP server exposes facts and deterministic derived state.

It does not need an internal language model.

The calling Codex, Claude Code, OpenCode, IDE agent, or another client performs natural-language reasoning using Specview as a source of structured context.

A future optional `Specview Agent` may provide higher-level interpretation, but it must consume the same public core/protocol contracts rather than gain privileged hidden state.

## A2A is the future agent-to-agent boundary

When Specview needs to participate as an independent agent that can receive delegated tasks, send messages, advertise capabilities, or collaborate with other remote agents, the preferred interoperability target is A2A rather than the legacy IBM/BeeAI ACP surface.

A2A is not required for the first read-only control-plane use case.

Potential later capabilities include:

```text
ask Specview to investigate abandoned sessions
request a repository activity summary
request a cross-host consistency check
subscribe an operations agent to blocked work
hand off an investigation between agents
```

These are agent-to-agent interactions, not merely tool calls.

## Agent Client Protocol is a separate optional adapter

Agent Client Protocol standardizes the boundary between an editor/IDE client and a coding agent.

It becomes relevant only if Specview exposes a first-class interactive `Specview Agent` that an IDE can launch or connect to as an agent session.

That is distinct from exposing Specview data to Codex/Claude/OpenCode through MCP.

Therefore Agent Client Protocol is not part of the first MCP slice and must not be confused with A2A/Agent Communication Protocol.

## Multi-host is a core topology problem, not a protocol problem

Running Specview on both a laptop and a devbox introduces host identity, repository identity, session identity, synchronization, conflict, freshness, and trust questions.

Those belong to the Specview host/federation model.

MCP, A2A, or Agent Client Protocol may expose federated state, but none of them should define Specview's canonical multi-host semantics.

The future topology is conceptually:

```text
Laptop Specview ─┐
                 ├── federated Specview projection
Devbox Specview ─┘
                         │
                         ├── UI
                         ├── CLI
                         ├── MCP
                         └── future A2A
```

A repository may be observed on multiple hosts simultaneously. Host and session identity must therefore remain explicit rather than being collapsed by repository name alone.

## Security direction

The first MCP server should default to the same local-first trust boundary as the web UI.

Remote exposure and multi-host federation require an explicit authentication and transport decision later. Specview must not silently expose repository paths, process/session metadata, Git state, or evidence over a network merely because a protocol adapter exists.

## Consequences

- MCP can deliver immediate value to existing coding agents with a small read-only adapter.
- Specview does not need an embedded LLM to become agent-accessible.
- A2A remains available when Specview becomes an independently addressable collaborating agent.
- Agent Client Protocol remains available for IDE-agent integration without polluting agent-to-agent semantics.
- multi-host federation can evolve independently from protocol transports.
- all interfaces reuse one normalized state model.

## Planned slices

```text
H18 - Read-only MCP server
H19 - Host identity and federation contract
H20 - Multi-host projection/synchronization POC
H21 - Additional execution adapters
later - A2A Specview Agent adapter
optional - Agent Client Protocol adapter
```

Numbering after H17 may be adjusted if another product slice lands first, but the dependency direction remains MCP before agent-to-agent orchestration.

## Non-goals

This ADR does not implement MCP, A2A, Agent Client Protocol, remote authentication, a cloud control plane, multi-host synchronization, or an embedded language model.
