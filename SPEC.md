# Specview v0.0.1 Release Candidate Specification

## Purpose

Specview is a local-first, read-only control plane for observing agentic software development.

It does not own the work. It observes authoritative systems, normalizes their state, and projects that state into human- and agent-readable views.

The core model is:

```text
INTENT | EXECUTION | EVIDENCE | ACCEPTANCE
```

Source-control, Host, and federation context connect those planes without becoming their authority.

The product must remain useful when individual optional integrations are unavailable. A GitHub failure must not hide local Git state. An Evidence failure must not hide Intent. A Host index failure must not stop repository observation. An unreachable federation peer must not erase its last valid source snapshot or invent fresh facts.

## Product thesis

Specview answers four questions:

1. What software work exists?
2. What is actively executing against it?
3. What evidence exists for the current revision?
4. Is that revision acceptable under the repository policy?

It can answer those questions for one Host and project a conservative read-only view across known Hosts.

The UI is a projection of this normalized model. Kanban, list, hierarchy, and future graph views are representations, not domain models.

## Authority boundaries

Specview is deliberately not a project-management database.

```text
Repository filesystem  -> repo-native Intent
Execution adapters     -> live Execution
Git                    -> local source-control state
Forge adapters         -> remote provider context
Evidence adapters      -> verification observations
Acceptance policy      -> repository acceptance rules
Host catalog           -> local observation/history compatibility state
SQLite host index      -> rebuildable search projection
HostSnapshot           -> immutable source-Host observation
Federation peer cache  -> last-known remote observation
Federation runtime     -> derived polling/freshness/projection behavior
```

Rules:

- repository files remain authoritative for durable repo-native Intent;
- Git remains authoritative for local repository/worktree state;
- execution adapters remain authoritative for live agent execution;
- provider state degrades independently from local Git;
- Evidence is revision-scoped and never silently promoted into Acceptance;
- Acceptance is derived from policy plus Evidence for one revision;
- SQLite is disposable and rebuildable;
- every remote HostSnapshot remains attributable to its source Host;
- federation correlation and polling create derived views, not shared authority;
- Specview does not write tasks, branches, pull requests, specifications, Evidence, or remote Host state in v0.0.1.

## Domain graph

The normalized product model is graph-first:

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
    ├── RemoteObservation
    └── HostSnapshot
```

Relationships matter more than any one visualization. A repository may have multiple worktrees and execution sessions. A WorkItem may have multiple supporting artifacts and Evidence records. Remote Hosts remain separately authoritative even when repository instances correlate into one derived group.

## Intent

Repository specification adapters normalize repo-native artifacts into a common model.

Supported projection adapters in v0.0.1:

- native Specview;
- GitHub Spec Kit;
- OpenSpec.

Additional strong conventions may be detected without parser support. Detection must never pretend an unsupported convention is fully understood.

Normalized artifacts distinguish:

- artifact kind;
- Knowledge vs Work plane;
- Primary vs Supporting role;
- WorkItem identity;
- relations between artifacts.

Native Specview work items use Markdown with namespaced front matter:

```markdown
---
specview:
  status: in_progress
---

# Transactional Outbox
```

Native work statuses remain:

```text
new
in_progress
done
```

Unknown or malformed metadata remains visible as an error instead of disappearing.

## Execution

Execution is normalized behind an adapter contract.

```text
ExecutionAdapter
      ↓
ExecutionRegistry
      ↓
ExecutionSession
```

A logical execution session is not an operating-system process. One session may contain multiple helper process IDs. Process IDs are diagnostics, not durable cross-Host identity.

The first automatic adapters observe Codex and Claude Code on supported macOS/Linux hosts. Adapter failures are isolated so one unavailable agent adapter does not erase sessions produced by another.

Execution sessions carry at least:

- adapter;
- logical session ID;
- agent;
- cwd;
- repository root;
- worktree root when known;
- process IDs as diagnostics.

## Source control and provider context

Local Git and remote forge/provider context are separate projections.

Git observation includes, when available:

- repository remote;
- worktrees;
- branch or detached state;
- HEAD;
- dirty count;
- upstream;
- locally known ahead/behind state;
- last commit.

Specview does not implicitly fetch remotes merely to render a page.

GitHub is the first provider adapter and uses the local `gh` CLI. Provider failure must degrade independently from Git. GitHub checks remain provider context unless explicitly translated through an Evidence adapter.

## Evidence

Evidence is a passive, normalized observation layer.

The native bridge reads strict JSON records under:

```text
.specview/evidence/
```

Each Evidence record is revision-scoped and distinguishes:

- stable logical `check`;
- concrete `provider`;
- kind;
- result;
- observation and execution timestamps;
- optional summary and metrics.

Evidence producers remain external. Specview is not the test runner or CI engine.

## Acceptance

Acceptance is a derived policy result over Evidence for the current revision.

Repository policy can require named checks and explicitly allow selected skipped checks. Dirty worktrees fail closed where a trustworthy revision cannot be established.

Acceptance must remain explainable: the projection exposes which required checks passed, failed, are missing, or were allowed to skip.

## Host observation and local history

Running `specview` starts a Host-level observer. It discovers repositories from observed execution activity rather than crawling the full filesystem.

The Host page answers:

- which repositories were recently active;
- which are active now;
- which agents are active;
- which specification convention is detected;
- where the repository lives locally.

Observed history is kept outside repositories in the Specview Host state directory.

The JSON Host catalog remains a compatibility/history snapshot in v0.0.1. SQLite is a derived, rebuildable index used for search. Deleting the SQLite index must be safe.

Search can match repository identity/context without making SQLite the authority for rendered live state.

## Live web projection

The browser is server-driven and read-only.

Material changes use:

```text
material state change
      ↓
SSE `changed`
      ↓
fetch targeted HTML fragment
      ↓
replace live projection
```

Heartbeat-only observations must not cause unnecessary browser fragment refreshes.

The current slice intentionally uses browser-native EventSource, fetch, AbortController, History API, and HTML templates instead of requiring a SPA framework.

## Read-only MCP

Specview exposes normalized local control-plane facts through a read-only MCP stdio interface.

The MCP layer is an adapter over the same domain contracts used by the web projection. It must not create a second source of truth or gain write authority in v0.0.1.

Language-neutral fixtures protect the observable contracts so the implementation can be replaced without redefining product semantics.

The public execution shape uses a logical session `id`; `process_ids` remain optional diagnostics.

## Host identity and federation

Each Host has persistent local identity. Repository instances remain Host-scoped.

Federation is conservative and read-only:

```text
Host A snapshot ─┐
Host B snapshot ─┼─> derived multi-host projection
Host C snapshot ─┘
```

Remote observations never override the source Host's authority.

### Snapshot and correlation contract

v0.0.1 federation includes:

- persistent Host identity;
- deterministic RepositoryInstance identity;
- conservative repository correlation;
- explicit optional project identity;
- HostSnapshot v1;
- deterministic aggregation of source snapshots.

When several snapshots exist for one Host, the newest source `observed_at` wins. Equal-time conflicting snapshots fail explicitly. Repository groups are derived projection state and must not become canonical global repository identity.

### Transport and peers

The first network transport is localhost-first HTTP pull.

Host-level peer state supports:

- required expected Host ID pinning;
- source URL validation;
- credential references without persisted secret values;
- manual refresh;
- last-known valid snapshot preservation;
- source `observed_at` separate from local `retrieved_at`;
- freshness states: `fresh`, `stale`, `unreachable`, `never_retrieved`.

Credential values are resolved at request time and must not be persisted or exposed in error text.

### Federation runtime

`specview serve` starts a derived federation polling runtime alongside local execution observation.

The runtime:

- re-opens the Host-level peer registry each cycle so peer add/remove is observed without restart;
- refreshes configured peers through the existing H22 refresher and security rules;
- isolates peer failures;
- changes cached observations, not remote authority;
- notifies observers only when peer material changes, not for transport-only attempt timestamps.

The deterministic current multi-host projection is available through:

```text
specview federation status
```

Projection rules:

- the local HostSnapshot is freshly built for each projection read;
- `fresh`, `stale`, and `unreachable` peers with a cached valid snapshot contribute that snapshot to unchanged H20 aggregation;
- `never_retrieved` peers remain visible as Hosts but contribute no invented repository facts;
- unreachable never means inactive or zero sessions;
- remote freshness metadata never rewrites source HostSnapshot fields;
- H20 conservative correlation semantics remain unchanged.

v0.0.1 does not include automatic peer discovery, push synchronization, remote execution, remote writes, per-peer polling schedules, or a shared database.

## Configuration

Repository configuration lives in:

```text
.specview.yaml
```

The v1 contract currently supports repository identity, project root, Intent adapter configuration, Acceptance policy, and legacy server fields retained for compatibility.

Example with an explicit Acceptance policy:

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
  required:
    - unit-tests
    - lint
  allow_skipped:
    - lint

server:
  host: 127.0.0.1
  port: 7331
```

Repositories that do not define Acceptance policy can omit the `acceptance` section.

`project.id` is optional explicit cross-Host project identity. It must not be synthesized from personal or machine identity.

Host identity, federation peers, remote observation cache, and other Host-level federation state live outside repositories.

The repository-level `server` section is a compatibility artifact in v1, not a precedent for putting future Host settings in repository configuration. Its cleanup or migration requires an explicit configuration-contract change rather than an incidental removal.

## CLI surface

The product includes the Host observer plus explicit read-only/control-plane utilities introduced through H01-H23.

Core commands include:

```text
specview
specview serve
specview mcp
specview init
specview doctor
specview federation snapshot
specview federation aggregate
specview federation serve
specview federation pull
specview federation peer ...
specview federation status
specview version
specview help
```

Peer mutation changes only local peer configuration/cache; it does not mutate a remote Host.

## Security and privacy

Default network exposure is loopback-first.

Secrets must not be persisted in federation peer files or error text. Peer credential configuration stores references, such as environment-variable names, rather than secret values.

Host state directories and sensitive local indexes use private filesystem permissions.

Repository observation must not write product state into unrelated repositories. Specview-owned ephemeral/runtime files under `.specview/` must remain clearly separated from durable repository Intent.

## Distribution

GitHub Releases are the v0.0.1 binary distribution origin.

Release artifacts target:

- Linux amd64;
- Linux arm64;
- macOS amd64;
- macOS arm64;
- SHA-256 checksums.

POC installer:

```bash
curl -fsSL https://raw.githubusercontent.com/sergii/specview/main/install.sh | sh
```

Future canonical installer:

```bash
curl -fsSL https://specview.sh/install | sh
```

## Release boundary

v0.0.1 is a proof of the observation/control-plane architecture, not a promise to complete every possible agentic-development feature.

Included product capabilities are those already defined and accepted by H01-H23, plus H24 release-stabilization work that does not expand the product surface.

Before v0.0.1, development is frozen against new feature planes. H24 may:

- correct contract/documentation drift;
- fix correctness, safety, portability, performance, or packaging defects;
- improve tests and acceptance gates;
- remove accidental coupling when it can be done without destabilizing frozen public contracts.

H24 must not add a new workflow, provider, agent family, federation transport/mode, UI paradigm, remote-write capability, or orchestration responsibility.

## Explicit non-goals for v0.0.1

- becoming a task/project management system;
- writing specification status from the UI;
- generating specifications as product authority;
- orchestrating agents;
- remote execution;
- remote repository writes;
- GitHub write operations;
- automatic federation peer discovery;
- push federation synchronization;
- per-peer polling schedules;
- shared multi-host database;
- semantic/vector search as a required dependency;
- coupling the domain model to Kanban, list, hierarchy, or graph presentation;
- requiring React, Vue, or another SPA framework.

## Definition of done

v0.0.1 is release-ready when:

1. canonical product documentation describes the architecture actually implemented through H23;
2. H01-H23 accepted contracts remain green;
3. formatting, module verification, vet, race tests, coverage, build, MCP/federation built-binary smoke tests, federation runtime/status smoke, browser semantic E2E, and release cross-build pass on the release candidate;
4. macOS and Linux release artifacts are reproducible from the release workflow;
5. installation from a GitHub Release works as a user installation, not only from a development checkout;
6. Host observation survives restart and does not require a repository-local Specview config to start;
7. Intent, Execution, Git/provider, Evidence, Acceptance, MCP, and federation failures degrade according to their documented independent authority boundaries;
8. no credential secret is persisted by federation peer state;
9. the release contains no hidden remote-write or agent-orchestration path;
10. unresolved architectural debt is explicitly classified as either release-blocking or post-v0.0.1 instead of being silently expanded into new feature work.
