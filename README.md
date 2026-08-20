# Specview

**Specview** is a small, read-only dashboard for observing Markdown specifications as they move through an agentic software-development workflow.

Website: **specview.sh**

> The domain is the canonical product home. During the proof of concept, binaries are distributed through GitHub Releases.

See [`SPEC.md`](SPEC.md) for the canonical v0.0.1 POC specification.

## Architecture

Specview observes a repository. The filesystem is the source of truth. Specview does not edit specifications or manage their state.

```text
agent / developer
      -> edits specs/*.md
      -> Specview observes
      -> live dashboard updates
```

The demo is intentionally a separate Git repository. Demo files, implementation code, and Git history are not embedded into the Specview binary.

## Try the demo

Canonical companion repository:

```text
https://github.com/specview/specview-demo
```

Once the Specview GitHub organization is available:

```bash
git clone https://github.com/specview/specview-demo.git
cd specview-demo
specview
```

The demo repository is a real project with its own `.git`, `.specview.yaml`, `specs/`, implementation, and history. Its config sets:

```yaml
project:
  name: "Demo Project"
  demo: true
```

Specview only understands the generic `project.demo` flag so the UI can display a small `DEMO` marker. It does not know or ship the demo dataset itself.

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

`project.name` is optional. When empty, Specview uses the repository directory name.

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
- optional `DEMO` marker driven by config
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
- optional generic demo marker
- `specs/` by default
- `new`, `in_progress`, `done`
- recursive Markdown discovery
- filesystem observation with a lightweight polling snapshot (250 ms)
- Markdown source preview
- live browser refresh over SSE
- metadata-error visibility
- loopback HTTP server by default
- GitHub Actions CI
- macOS/Linux release archives for amd64/arm64
- SHA-256 checksums
- `install.sh`
- external companion demo repository reference

Explicitly not included:

- embedded demo files
- automatic demo cloning
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
