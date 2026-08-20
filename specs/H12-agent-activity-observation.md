---
specview:
  status: in_progress
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

## POC filesystem transport

The current POC observes local runtime records at:

```text
.specview/runtime/activity/<session-id>.json
```

The runtime directory is ephemeral project-local state and should be ignored by Git.

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

`spec` is project-relative and therefore includes the configured specs directory such as `specs/`.

Only JSON records with `version: 1`, a session ID, spec path, state, and heartbeat are projected. Invalid activity files are ignored for presentation and logged as warnings instead of breaking the specification observer.

The filesystem transport is intentionally replaceable. A local socket, process adapter, MCP integration, or other low-overhead presence source may later complement it while preserving the same semantic contract.

## Agent identity

Do not pretend agent identity can always be inferred reliably from filesystem edits alone.

Prefer explicit metadata supplied by the integration that owns the session:

- stable machine-readable `agent.id` such as `codex`, `claude-code`, `opencode`, or another opaque provider identifier.
- human-readable `agent.label` for presentation.
- opaque `session_id` so multiple concurrent sessions from the same agent type remain distinct.

Process-name or environment inspection may be used only as a best-effort adapter technique, not as canonical identity.

Unknown agents remain valid and render as `Agent` rather than being guessed incorrectly.

## Multiple agents

More than one agent may work on the same specification or on different specifications concurrently.

The projection supports:

```text
Spec A -> Codex
Spec B -> Claude Code
Spec C -> Codex + OpenCode
```

A single active session renders its agent label. Multiple live sessions on one specification collapse to `<n> agents` in the compact board presentation.

## Liveness

Activity is valid only while its heartbeat is fresh.

The current POC uses a **30 second TTL**. A `working` session is active while its heartbeat remains inside that window. When the heartbeat expires, Specview detects the active-session signature change and sends one SSE refresh, so a dead session disappears even if the agent never performs cleanup.

Filesystem changes to activity records use the same 250 ms polling watcher mechanism as the current specification POC and also trigger SSE refresh.

## Visual language

Agent activity is an overlay, not a fourth workflow status.

Current presentation:

- workflow state keeps the square markers for New, In progress, and Done.
- active work uses a tiny animated square-outline glyph with an asymmetric filled corner.
- the asymmetry makes rotation visible while preserving the square visual language.
- the agent label sits beside the glyph and beside the existing relative modification age.
- Classic, Dense, and Flow render the same activity semantics without changing workflow state.
- animation respects `prefers-reduced-motion` and becomes static when reduced motion is requested.

Do not animate inactive or stale specifications.

## Privacy and paths

Presence records are local runtime state and should normally be ignored by Git.

Do not include prompts, conversation content, credentials, user identity, hostnames, or unrelated process metadata merely to identify an active agent.

Only publish the minimum information required to correlate a session with a specification and render useful activity state.

## Publisher adapters still needed

The observer side is now implemented, but producers remain future work.

Useful adapters include:

- Codex wrapper / hook.
- Claude Code hook.
- OpenCode integration.
- generic shell helper for any agent that can periodically rewrite one heartbeat JSON file.

Adapters should generate opaque session IDs, publish explicit agent identity, update heartbeat atomically, and remove their record on clean exit when possible. Correct cleanup is an optimization; TTL expiry remains the reliability mechanism.

## Acceptance criteria

- workflow status and live agent activity remain separate concepts.
- `in_progress` alone never produces a working spinner.
- Specview remains read-only.
- agent identity is explicit when available and neutral when unknown.
- multiple simultaneous agent sessions are representable.
- stale sessions disappear from active presentation automatically after a 30 second heartbeat TTL.
- runtime presence does not require repeated Markdown edits.
- activity filesystem changes participate in SSE live refresh.
- invalid runtime records do not break the specification dashboard.
- Flow, Dense, and Classic render the same activity projection without changing its semantics.
- publisher adapters can be added without changing the observer contract.
