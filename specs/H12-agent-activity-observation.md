---
specview:
  status: new
---

# Agent activity observation

Add an optional, ephemeral activity layer that lets Specview show which agent is actively working on a specification without confusing workflow state with runtime presence.

## Core distinction

Specification status and agent activity are different facts.

```text
specview.status = durable workflow state
agent activity  = ephemeral runtime presence
```

`in_progress` means the specification belongs to the In progress workflow state. It does **not** prove that an agent is executing work right now.

Specview must therefore never show a working spinner solely because a specification has `status: in_progress`.

## Read-only boundary

Specview remains read-only.

It does not claim tasks, edit specifications, start agents, or write activity state. An agent, wrapper, IDE integration, or adapter may publish ephemeral presence for Specview to observe.

The durable specification remains Markdown. Runtime activity must not require rewriting Markdown front matter on every heartbeat because that would create repository churn, noisy filesystem events, and misleading Git history.

## Presence contract

A future implementation may observe ignored local runtime records such as:

```text
.specview/runtime/activity/<session-id>.json
```

Example:

```json
{
  "version": 1,
  "session_id": "01K...",
  "agent": {
    "id": "codex",
    "label": "Codex"
  },
  "spec": "specs/H10-thin-editorial-kanban.md",
  "state": "working",
  "started_at": "2026-08-20T18:00:00Z",
  "heartbeat_at": "2026-08-20T18:00:12Z"
}
```

The exact transport is not fixed by this spec. A local socket, process adapter, MCP integration, or other low-overhead presence source may later replace or complement files while preserving the same semantic contract.

## Agent identity

Do not pretend agent identity can always be inferred reliably from filesystem edits alone.

Prefer explicit metadata supplied by the integration that owns the session:

- stable machine-readable `agent.id` such as `codex`, `claude-code`, `opencode`, or another opaque provider identifier.
- human-readable `agent.label` for presentation.
- opaque `session_id` so multiple concurrent sessions from the same agent type remain distinct.

Process-name or environment inspection may be used only as a best-effort adapter technique, not as canonical identity.

Unknown agents remain valid and should render as `Agent` or another neutral label rather than being guessed incorrectly.

## Multiple agents

More than one agent may work on the same specification or on different specifications concurrently.

The projection must support:

```text
Spec A -> Codex
Spec B -> Claude Code
Spec C -> Codex + OpenCode
```

The UI may collapse multiple sessions into `2 agents` when space is limited while preserving the full session list for a future detail view.

## Liveness

Activity is valid only while its heartbeat is fresh.

A future implementation should define a short liveness TTL. When the latest heartbeat expires, the session becomes stale and must stop rendering as actively working without requiring the agent to perform cleanup perfectly.

The TTL should be long enough to tolerate brief pauses and filesystem scheduling delays, but short enough that a dead process does not appear active for minutes.

## Visual language

Agent activity is an overlay, not a fourth workflow status.

Suggested presentation:

- keep the square workflow markers for New, In progress, and Done.
- show active work with a tiny animated **square-outline activity glyph**, not a large circular loading spinner.
- rotate or step the square subtly while the heartbeat is live.
- place a quiet agent label beside it, for example `Codex`, `Claude Code`, or `2 agents`.
- do not animate inactive or stale specifications.
- Classic may show activity in card metadata.
- Dense may show it as a compact trailing field.
- Flow is the most natural mode for continuously changing activity because it has almost no surrounding chrome.

Animation must respect `prefers-reduced-motion`.

## Privacy and paths

Presence records are local runtime state and should normally be ignored by Git.

Do not include prompts, conversation content, credentials, user identity, hostnames, or unrelated process metadata merely to identify an active agent.

Only publish the minimum information required to correlate a session with a specification and render useful activity state.

## Acceptance criteria

- workflow status and live agent activity remain separate concepts.
- `in_progress` alone never produces a working spinner.
- Specview remains read-only.
- agent identity is explicit when available and neutral when unknown.
- multiple simultaneous agent sessions are representable.
- stale sessions disappear from active presentation automatically after a heartbeat TTL.
- runtime presence does not require repeated Markdown edits.
- Flow, Dense, and Classic can render the same activity projection differently without changing its semantics.
