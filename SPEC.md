# Specview v0.0.1 POC Specification

## Purpose

Specview is a read-only observer for Markdown specifications used during fast, agentic software-development workflows.

The filesystem is the source of truth. Specview watches specification files and renders their current state as a live local dashboard. It does not own tasks, edit specifications, generate specifications, or write status changes.

## Repository model

Specview and its demo are separate repositories.

Canonical target layout:

```text
github.com/specview/specview
github.com/specview/specview-demo
```

Responsibilities:

- `specview/specview` contains the Go binary, dashboard, configuration contract, installer, CI, and release workflows.
- `specview/specview-demo` is a normal Git repository used to demonstrate Specview with realistic specifications and implementation code.
- the Specview binary must not embed or vendor the demo specifications.
- the demo repository has independent Git history and may evolve independently from Specview releases.

During the initial bootstrap, the implementation repository may temporarily live under another owner before being transferred to the Specview organization.

## Vertical slice

The v0.0.1 proof of concept must support this complete path:

```text
install Specview
    -> enter a repository
    -> read .specview.yaml
    -> discover specs/**/*.md
    -> parse specview.status
    -> render three-column dashboard
    -> detect filesystem change
    -> refresh in-memory projection
    -> notify browser over SSE
    -> show the updated state
```

The demo validates the same path by being cloned as an ordinary repository and observed with the same binary.

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
- when `project.name` is empty, the repository directory name is used.
- `project.demo: true` is an optional generic presentation hint.
- `project.demo` does not enable hidden data, network access, cloning, or special observation behavior.
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
- optional small `DEMO` marker when `project.demo: true`

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

## Demo repository

The companion repository is:

```text
https://github.com/specview/specview-demo
```

It is a real project, not a fixture embedded in the Specview binary.

Expected structure:

```text
specview-demo/
├── .git/
├── .specview.yaml
├── README.md
├── specs/
│   ├── 01-project-setup.md
│   ├── ...
│   └── 10-release.md
└── implementation files
```

Initial dataset:

- exactly 10 specifications
- 4 `new`
- 3 `in_progress`
- 3 `done`
- a small but real implementation so specs can correspond to code changes

Demo configuration:

```yaml
version: 1

project:
  name: "Demo Project"
  demo: true

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
```

Primary demo flow:

```bash
git clone https://github.com/specview/specview-demo.git
cd specview-demo
specview
```

Users must be able to edit the demo specs, observe live transitions, commit changes, reset the repository, branch it, or use an AI coding agent inside it exactly like any other Git project.

The main Specview repository should reference this companion repository in its README and documentation but must not copy its dataset into release binaries.

## CLI

Required commands:

```text
specview
specview serve
specview init
specview version
specview help
```

There is intentionally no special `specview demo` or `specview init --demo` command in v0.0.1. Demo usage is ordinary Git usage.

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
- Git integration inside Specview
- GitHub API integration inside Specview
- ontology or semantic validation
- embedded demo dataset
- automatic demo repository cloning

## Definition of done

The POC is complete when:

1. a user can install the Specview binary;
2. a user can initialize any repository with `specview init` and observe its specs;
3. the external `specview-demo` repository can be cloned normally;
4. running `specview` inside the demo repository shows Demo Project with 10 specifications;
5. editing one specification status moves its card automatically without manual browser refresh;
6. the Specview release binary contains no demo specification dataset.
