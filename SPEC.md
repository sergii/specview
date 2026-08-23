# Specview v0.0.1 POC Specification

## Purpose

Specview is a read-only observer for Markdown specifications used during fast, agentic software-development workflows.

The filesystem is the source of truth. Specview watches specification files and renders their current state as a live local dashboard. It does not own tasks, edit specifications, generate specifications, or write status changes.

## Product repository

The canonical product repository is:

```text
github.com/sergii/specview
```

It contains the Go binary, dashboard, configuration contract, installer, CI, release workflows, product documentation, the canonical demo recipe, and the real specifications used to build Specview itself.

The separate `sergii/specview-demo` repository may remain temporarily as a development/integration fixture. It is not required by the product architecture or public demo flow.

## Vertical slice

The v0.0.1 proof of concept must support this complete path:

```text
install Specview
    -> enter a project or load its configuration
    -> resolve project.root
    -> discover configured specs/**/*.md
    -> parse specview.status
    -> render three-column dashboard
    -> detect filesystem change
    -> refresh in-memory projection
    -> notify browser over SSE
    -> show the updated state
```

## Configuration

Default file: `.specview.yaml`.

```yaml
version: 1

project:
  name: ""
  root: "."

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
```

Rules:

- `project.name` is optional.
- when `project.name` is empty, the observed project directory name is used.
- `project.root` defaults to `.` when omitted.
- relative `project.root` values are resolved from the directory containing `.specview.yaml`.
- absolute `project.root` values are allowed.
- `specs.path` is resolved from the observed project root.
- `specs.path` remains relative.
- the server binds to loopback by default.

`project.root` is a generic filesystem capability. It is not a demo switch and does not imply Git.

## Specification contract

A specification is a Markdown file discovered under the configured specs directory.

Status metadata:

```yaml
---
specview:
  status: in_progress
---
```

Valid statuses for v0.0.1:

1. `new`
2. `in_progress`
3. `done`

A file without Specview metadata defaults to `new`.

An invalid or unknown status must remain visible and appear under Metadata errors. Specview must never silently discard a specification because its metadata is invalid.

## Observation model

Specview is read-only.

```text
agent / developer
      -> edits Markdown
      -> filesystem changes
      -> Specview observes
      -> dashboard changes
```

The browser must not mutate a specification or status.

For the POC, filesystem changes may be detected by a 250 ms polling snapshot. The implementation may later move to native filesystem notifications without changing the external contract.

## Graceful shutdown

SIGINT and SIGTERM must stop Specview cleanly even while one or more browser clients keep the SSE endpoint open.

The first `Ctrl+C` must terminate the process without waiting for the shutdown timeout and without printing `context deadline exceeded`.

## Dashboard

The dashboard is intentionally minimal.

Required elements:

- Specview product name
- project name
- configured specs path
- total spec count
- live indicator
- New column
- In progress column
- Done column
- count per column
- specification cards with title, path, and modified age
- Metadata errors section when needed
- specification detail view

Explicitly avoid in v0.0.1:

- sidebar navigation
- filters
- drag and drop
- status editing
- comments
- assignees
- avatars
- sprint concepts
- Jira/Linear-style project-management controls

The UI should feel like an activity/observation surface, not a task manager.

## Ephemeral demo

The canonical demo is an agent-executable scenario stored in:

```text
demo.md
```

Concept:

```text
Specview Demo is a reproducible scenario, not persistent data.
```

The agent must create a unique temporary directory using the operating system's standard temporary-directory mechanism with the prefix:

```text
specview-demo-
```

The resulting semantic name is:

```text
specview-demo-<opaque-session-id>
```

The suffix is opaque. It must not be derived from or encode:

- username
- hostname
- laptop or device name
- repository name
- Git SHA
- email address
- other personal or machine identity

Each demo session creates its own `.specview.yaml` and `specs/` inside the temporary directory.

The initial demo state contains six specs:

- 2 `done`
- 2 `in_progress`
- 2 `new`

### Demo lifecycle

```text
prepare
  -> create unique temporary directory
  -> create .specview.yaml
  -> create six specs
  -> print path and observation command
  -> wait

run demo
  -> perform visible state transitions
  -> pause between transitions
  -> print each transition
  -> keep temporary state after completion

cleanup
  -> only after explicit user instruction
  -> remove the exact temporary directory for that session
```

The agent must not modify the user's current project, initialize Git, make commits, or write outside the temporary demo directory.

Multiple demo sessions must be able to run concurrently without sharing state.

The Specview binary must contain no demo-specific behavior, bundled fixtures, demo cloning, or cleanup logic.

## Dogfooding

The Specview source repository is itself configured as a Specview project:

```text
sergii/specview/
├── .specview.yaml
├── specs/
├── cmd/
└── internal/
```

This is the real-world example. `demo.md` is the deterministic synthetic demonstration.

## CLI

Required commands:

```text
specview
specview serve
specview init
specview version
specview help
```

There is intentionally no special `specview demo`, `specview --demo`, or `specview init --demo` command in v0.0.1.

A generic path-oriented CLI such as `specview [path]` may be added separately because it is useful beyond demo scenarios.

## Git status observation

Showing Git state is a natural next observation capability, but it is outside the v0.0.1 vertical slice. Specview's core filesystem observation must not require Git.

## Distribution

During the POC, GitHub Releases are the binary distribution origin.

Release artifacts:

- Linux amd64
- Linux arm64
- macOS amd64
- macOS arm64
- `SHA256SUMS`

POC installer:

```bash
curl -fsSL https://raw.githubusercontent.com/sergii/specview/main/install.sh | sh
```

Future canonical installer:

```bash
curl -fsSL https://specview.sh/install | sh
```

## Non-goals

v0.0.1 does not include:

- database
- authentication
- remote service
- write API
- task management
- specification generation
- LLM integration inside Specview
- Git status integration
- GitHub API integration inside Specview
- ontology or semantic validation
- embedded demo dataset
- persistent demo state as a product requirement
- automatic demo repository cloning
- demo-specific binary modes

## Definition of done

The POC is complete when:

1. a user can install the Specview binary;
2. a user can initialize a project with `specview init` and observe its specs;
3. `project.root` defaults to `.` and can point to another filesystem root;
4. editing one specification status moves its card automatically without manual browser refresh;
5. `Ctrl+C` stops Specview cleanly with an active SSE client;
6. an agent can read `demo.md`, create an isolated ephemeral demo, and pause for observation;
7. `run demo` produces visible status transitions with pauses;
8. `cleanup` removes only that demo session's temporary directory;
9. the Specview release binary contains no demo specification dataset or demo-specific behavior;
10. the Specview source repository can be observed using its own `.specview.yaml` and `specs/`.
