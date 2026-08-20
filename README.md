# Specview

**Specview** is a small, read-only dashboard for observing Markdown specifications as they move through an agentic software-development workflow.

Website: **specview.sh**

> The domain is the canonical product home. During the proof of concept, binaries are distributed through GitHub Releases.

See [`SPEC.md`](SPEC.md) for the canonical v0.0.1 POC specification.

## Try the demo

After installing Specview, the fastest first-run experience is:

```bash
specview demo
```

This starts an isolated **Demo Project** with 10 bundled specifications and does not change the current repository. The dashboard marks the project with a small `DEMO` badge.

The demo contains:

- 4 `new` specs
- 3 `in_progress` specs
- 3 `done` specs

To copy the same fixture into a fresh repository instead:

```bash
specview init --demo
specview
```

Then edit any file under `specs/`, for example change `status: new` to `status: in_progress`. The card moves automatically without manually refreshing the browser.

## Install

After the first GitHub Release is published:

```bash
curl -fsSL https://raw.githubusercontent.com/sergii/specview/main/install.sh | sh
```

The installer detects macOS/Linux and amd64/arm64, downloads the matching release archive, verifies its SHA-256 checksum, and installs `specview` to `~/.local/bin` by default.

The future canonical installer will be:

```bash
curl -fsSL https://specview.sh/install | sh
```

## Initialize a repository

```bash
cd your-repository
specview init
```

This creates:

```text
.specview.yaml
specs/
```

Default configuration:

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

`project.name` is optional. When empty, Specview uses the repository directory name. Demo projects additionally set `project.demo: true` so the UI can identify fixture data without changing the observation model.

The leading dot in `.specview.yaml` is intentional: it is tooling metadata, not project documentation.

## Specification status contract

Specifications are regular Markdown files. Status is namespaced under `specview` in YAML front matter:

```markdown
---
specview:
  status: in_progress
---

# Transactional Outbox

Implementation notes go here.
```

The POC has exactly three valid statuses:

```text
new
in_progress
done
```

A specification without front matter defaults to `new`. An unknown status remains visible in the dashboard under **Metadata errors** instead of being silently ignored.

## Observe

```bash
specview
```

or:

```bash
specview serve
```

Open:

```text
http://127.0.0.1:7331
```

Specview watches `specs/**/*.md`. Create, edit, rename, or delete a specification and the browser updates automatically through Server-Sent Events.

For a remote devbox:

```bash
ssh -L 7331:127.0.0.1:7331 your-devbox
```

Then open `http://127.0.0.1:7331` locally.

## Dashboard philosophy

The POC UI is deliberately minimal:

- one header
- project name and specs path
- three columns: New, In progress, Done
- simple specification cards
- metadata errors when present
- one detail view
- no sidebar
- no filters
- no drag and drop
- no task-management controls

The dashboard is a read-only projection of filesystem state, not another project-management system.

## Scope of v0.0.1

Included:

- one Go binary
- `.specview.yaml`
- optional project name
- `specs/` by default
- `new`, `in_progress`, `done`
- recursive Markdown discovery
- filesystem observation with a lightweight polling snapshot (250 ms)
- Markdown source preview
- live browser refresh over SSE
- metadata-error visibility
- `specview demo`
- `specview init --demo`
- 10 bundled demo specifications
- minimal `DEMO` indication in the UI
- loopback HTTP server by default
- GitHub Actions CI
- macOS/Linux release archives for amd64/arm64
- SHA-256 checksums
- `install.sh`

Explicitly not included:

- database
- authentication
- write API
- task management
- spec generation
- Git integration
- GitHub integration
- AI/LLM calls

## Development

CI and release builds use Go 1.26.x; the POC source intentionally stays compatible with Go 1.23+ and has no runtime or module dependencies.

```bash
go test ./...
go vet ./...
go build ./cmd/specview
```

## Releases

Normal releases can be created by pushing a `v*` tag. For the first POC, the **Release** workflow can also be started manually with a version such as `v0.0.1`; it creates the GitHub Release and attaches all four platform archives plus `SHA256SUMS`.
