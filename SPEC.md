# Specview v0.0.1 POC Specification

## Purpose

Specview is a read-only observer for Markdown specifications used during fast, agentic software-development workflows.

The filesystem is the source of truth. Specview watches specification files and renders their current state as a live local dashboard. It does not own tasks, edit specifications, generate specifications, or write status changes.

## Vertical slice

The v0.0.1 proof of concept must support this complete path:

```text
install Specview
    -> initialize or start demo
    -> read .specview.yaml
    -> discover specs/**/*.md
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

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
```

Rules:

- `project.name` is optional.
- When `project.name` is empty, the repository directory name is used.
- `project.demo: true` may be used by bundled fixture projects to mark demo data in the UI.
- `specs.path` is relative to the repository root.
- the server binds to loopback by default.

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

## Demo Project

The repository contains a bundled `demo/` fixture with exactly 10 specifications.

Distribution:

- 4 `new`
- 3 `in_progress`
- 3 `done`

### Isolated demo

```bash
specview demo
```

Runs an isolated temporary **Demo Project** and must not modify the current repository.

### Demo data in the current repository

```bash
specview init --demo
```

Initializes the current uninitialized repository with `.specview.yaml` and the 10 demo specifications. It must refuse to overwrite an existing `.specview.yaml` or a conflicting demo spec file.

Demo configuration:

```yaml
project:
  name: "Demo Project"
  demo: true
```

When `project.demo` is true, the dashboard shows a small `DEMO` badge and a short note telling the user to edit specification status to see live updates. Demo styling must remain visually subordinate to the normal dashboard.

The demo fixture also serves as a deterministic integration-test dataset.

## CLI

Required commands:

```text
specview
specview serve
specview init
specview init --demo
specview demo
specview version
specview help
```

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
- LLM integration
- Git integration
- GitHub integration
- ontology or semantic validation

## Definition of done

The POC is complete when a user can install the binary, run `specview demo`, see Demo Project with 10 cards, change a demo spec from one valid status to another, and observe the card move to the correct column automatically without manually refreshing the browser.
