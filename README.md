# Specview

**Specview** is a local-first, read-only control plane for observing agentic software development.

Website: **specview.sh**

The canonical v0.0.1 architecture and release boundary live in [`SPEC.md`](SPEC.md).

## What Specview observes

Specview normalizes four orthogonal planes:

```text
INTENT | EXECUTION | EVIDENCE | ACCEPTANCE
```

It also projects the Git, forge/provider, Host, and federation context that connects those planes.

Specview does not own tasks, edit specifications, run your tests, orchestrate agents, or write remote repositories. It observes authoritative systems and renders read-only human and agent views of their state.

A simplified model:

```text
Host
├── Repository
│   ├── Worktree
│   │   └── ExecutionSession
│   ├── WorkItem
│   │   ├── Artifact
│   │   ├── Evidence
│   │   └── Acceptance
│   └── SourceControl
│       ├── Git
│       └── Provider
└── FederationPeer
    └── HostSnapshot
```

Kanban, list, hierarchy, and future graph views are projections over this model. The visualization is not the source of truth.

## Current v0.0.1 capabilities

- Host-level repository activity observation
- Codex and Claude Code execution adapters on supported macOS/Linux hosts
- normalized logical execution sessions with process IDs kept as diagnostics
- native Specview, GitHub Spec Kit, and OpenSpec Intent adapters
- read-only detection of additional strong specification conventions
- local Git/worktree context
- GitHub provider context through the local `gh` CLI
- normalized revision-scoped Evidence
- Acceptance policy derived from Evidence
- rebuildable SQLite Host search index
- server-driven live web UI over SSE and targeted HTML fragments
- read-only MCP stdio server
- persistent Host identity
- HostSnapshot v1 federation contract
- localhost-first federation HTTP pull transport
- Host-level federation peer registry with freshness and cached last-known observations
- periodic peer refresh while `specview serve` is running
- deterministic local + remote federation status projection
- language-neutral contract fixtures and built-binary smoke gates

## Authority model

Specview deliberately keeps authorities separate:

```text
Repository filesystem  -> durable repo-native Intent
Execution adapters     -> live Execution
Git                    -> local source-control state
Forge adapters         -> remote provider context
Evidence adapters      -> verification observations
Acceptance policy      -> acceptance rules
Host catalog           -> local history/compatibility snapshot
SQLite                  -> rebuildable search projection
Remote HostSnapshot     -> read-only remote Host observation
Federation runtime      -> derived refresh/projection behavior
```

If an optional source fails, unrelated facts should remain visible. GitHub failure, for example, must not hide local Git state. Federation polling changes cached remote observations but never upgrades them into shared authority.

## Install

During the POC, binaries are distributed through GitHub Releases.

```bash
curl -fsSL https://raw.githubusercontent.com/sergii/specview/main/install.sh | sh
```

The installer detects macOS/Linux and amd64/arm64, verifies the release checksum, and installs to `~/.local/bin` by default.

The future canonical installer will be:

```bash
curl -fsSL https://specview.sh/install | sh
```

## Start the Host dashboard

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

The Host dashboard does not require `.specview.yaml` in the current directory. Specview discovers repositories from observed agent execution rather than crawling the full filesystem.

For a remote devbox, keep the observer loopback-only and tunnel it:

```bash
ssh -L 7331:127.0.0.1:7331 your-devbox
```

## Initialize repository Intent

```bash
cd your-project
specview init
```

With no stronger supported convention, Specview creates:

```text
.specview.yaml
specs/
```

The current v1 configuration contract supports repository identity, Intent configuration, Acceptance policy, and legacy repository-level server fields:

```yaml
version: 1

project:
  id: ""
  name: ""
  root: "."

specs:
  adapter: specview
  path: specs
  pattern: "*.md"

acceptance:
  required: []
  allow_skipped: []

server:
  host: 127.0.0.1
  port: 7331
```

`project.id` is optional explicit cross-Host project identity. It must not be synthesized from personal or machine identity.

The repository-level `server` section is retained as a v1 compatibility field. Future Host-level settings belong in Host-level configuration or explicit CLI/environment configuration, not in repository Intent configuration by default.

## Native specification status

Native Specview work items are regular Markdown files:

```markdown
---
specview:
  status: in_progress
---

# Transactional Outbox

Implementation notes go here.
```

The native v0.0.1 statuses are:

```text
new
in_progress
done
```

Missing status defaults to `new`. Invalid metadata stays visible as an error instead of being silently discarded.

## Intent adapters

Specview normalizes different repo-native conventions into the same artifact/work-item model.

Supported adapters:

- `specview`
- `github-spec-kit`
- `openspec`

The normalized model distinguishes artifact kind, Knowledge vs Work, Primary vs Supporting role, WorkItem identity, and artifact relations.

## Execution

Execution is normalized behind:

```text
ExecutionAdapter
      ↓
ExecutionRegistry
      ↓
ExecutionSession
```

A logical execution session can represent several helper OS processes. PID is diagnostic context, not durable logical session identity.

For diagnostics:

```bash
specview doctor
```

## Git and provider context

Repository views can expose:

- Git remote
- worktrees
- branch or detached state
- HEAD
- dirty count
- upstream
- locally known ahead/behind state
- last commit
- matching GitHub pull requests and check summaries

Specview does not implicitly `git fetch` just to render a page.

GitHub integration uses the locally authenticated `gh` CLI, so Specview does not need to store separate GitHub credentials.

Provider checks remain provider context. They are not silently promoted into normalized Evidence.

## Evidence

The native Evidence bridge observes strict JSON records under:

```text
.specview/evidence/
```

Evidence is revision-scoped and separates a stable logical `check` from the concrete `provider` that produced it.

Specview remains passive. CI, test runners, linters, security tools, and other producers create Evidence; Specview observes it.

## Acceptance

Acceptance is derived from repository policy plus Evidence for one revision.

Example:

```yaml
acceptance:
  required:
    - unit-tests
    - lint
  allow_skipped:
    - lint
```

Every `allow_skipped` check must also appear in `required`.

Dirty worktrees fail closed where a trustworthy revision cannot be established. Evidence for an older or clean revision cannot make modified local work accepted.

## Read-only MCP

Run the MCP server over stdin/stdout:

```bash
specview mcp
```

The MCP interface exposes the same normalized control-plane facts used by the web projection. It is read-only in v0.0.1.

The public execution shape exposes a logical session `id` and keeps `process_ids` as diagnostics.

## Federation

Write the current Host snapshot:

```bash
specview federation snapshot > laptop.json
```

Aggregate snapshots:

```bash
specview federation aggregate laptop.json devbox.json
```

Serve the local snapshot endpoint:

```bash
specview federation serve
```

The built-in HTTP server binds to `127.0.0.1:7332`. A private network layer such as Tailscale Serve can publish it without changing Specview's loopback-first default.

Pull and validate a remote snapshot:

```bash
specview federation pull --expect-host host:550e8400-e29b-41d4-a716-446655440000 https://devbox.example.ts.net
```

### Federation peers

Add a Host-pinned peer:

```bash
specview federation peer add devbox \
  --url https://devbox.example.ts.net \
  --host host:550e8400-e29b-41d4-a716-446655440000 \
  --stale-after 5m
```

Optionally resolve an HTTP credential header from an environment variable:

```bash
specview federation peer add devbox \
  --url https://devbox.example.ts.net \
  --host host:550e8400-e29b-41d4-a716-446655440000 \
  --header-env 'Authorization=SPECVIEW_DEVBOX_AUTH'
```

Only the environment-variable name is persisted. The credential value is resolved at request time and must not be written to peer state or surfaced in errors.

Peer lifecycle commands:

```bash
specview federation peer list
specview federation peer show devbox
specview federation peer refresh devbox
specview federation peer remove devbox
```

Freshness states are:

```text
fresh
stale
unreachable
never_retrieved
```

A failed refresh preserves the last valid remote HostSnapshot.

### Federation runtime

When `specview serve` runs, Specview periodically re-opens the Host-level peer registry and refreshes currently configured peers. Peer failures are isolated and transport-only timestamps do not create material Host notifications.

Read the deterministic current local + remote projection with:

```bash
specview federation status
```

The status projection includes the freshly built local Host plus configured remote Hosts and their freshness. Cached snapshots from unreachable peers remain attributable to their source Host. `never_retrieved` peers are visible without invented repository facts.

Federation remains read-only. There is no automatic peer discovery, push sync, remote execution, remote write path, per-peer scheduling, or shared database in v0.0.1.

## Live UI

The browser is intentionally server-driven:

```text
material state change
      ↓
SSE changed event
      ↓
fetch targeted HTML fragment
      ↓
replace live projection
```

This keeps the current runtime small and avoids introducing a SPA framework where browser-native primitives are sufficient.

## Demo

`demo.md` is the canonical agent-executable demo recipe. The demo is a reproducible scenario, not persistent product data.

An agent creates an isolated temporary project, prints its path, waits for observation, performs visible state transitions, and only cleans up after explicit instruction.

Specview itself contains no special demo mode or bundled demo dataset.

## Development

For normal development:

```bash
bin/dev install
```

or:

```bash
bin/install
```

Diagnostics:

```bash
bin/doctor
```

The current release gate includes formatting, module verification, vet, race tests, coverage, build, built-binary MCP/federation smoke tests, federation runtime/status smoke, Chromium semantic E2E, and release cross-builds.

Logging is documented in `docs/observability/logging.md`.

## v0.0.1 feature freeze

H23 completed the first federation runtime and deterministic multi-host status projection. H24 is the release-stabilization slice.

H24 may fix correctness, safety, portability, performance, documentation, packaging, and test gaps, but it must not add another product plane or major feature family before the first release.

See:

```text
SPEC.md
specs/H24-v001-release-stabilization.md
```

The next feature slice starts only after the v0.0.1 release head is green and the installed binary has passed real-user dogfooding.
