# Specview

**Specview** is a small, read-only dashboard for observing Markdown specifications as they move through an agentic software-development workflow.

Website: **specview.sh**

> The domain is the canonical product home. During the proof of concept, binaries are distributed through GitHub Releases.

## Proof of concept

Specview watches a `specs/` directory in the repository where it is started and presents a live local dashboard. The filesystem is the source of truth. Specview does not edit specifications or manage their state.

### Install

After the first GitHub Release is published:

```bash
curl -fsSL https://raw.githubusercontent.com/sergii/specview/main/install.sh | sh
```

The installer detects macOS/Linux and amd64/arm64, downloads the matching release archive, verifies its SHA-256 checksum, and installs `specview` to `~/.local/bin` by default.

Override the installation directory when needed:

```bash
SPECVIEW_INSTALL_DIR=/usr/local/bin sh -c "$(curl -fsSL https://raw.githubusercontent.com/sergii/specview/main/install.sh)"
```

The future canonical installer will be:

```bash
curl -fsSL https://specview.sh/install | sh
```

### Initialize a repository

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

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
```

The leading dot is intentional: `.specview.yaml` is tooling metadata, not project documentation.

### Specification status contract

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

### Observe

```bash
specview
```

or explicitly:

```bash
specview serve
```

Open:

```text
http://127.0.0.1:7331
```

Specview watches `specs/**/*.md`. Create, edit, rename, or delete a specification and the browser updates automatically through Server-Sent Events.

For a remote devbox, keep Specview bound to loopback and use a tunnel, for example:

```bash
ssh -L 7331:127.0.0.1:7331 your-devbox
```

Then open `http://127.0.0.1:7331` locally.

## Scope of v0.0.1

Included:

- one Go binary
- `.specview.yaml`
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

Build all release archives locally:

```bash
./scripts/build-release.sh v0.0.1
```

## Releases

Normal releases can be created by pushing a `v*` tag. For the first POC, the **Release** workflow can also be started manually with a version such as `v0.0.1`; it creates the GitHub Release and attaches all four platform archives plus `SHA256SUMS`.
