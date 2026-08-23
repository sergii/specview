# Specview v0.0.1 Release Candidate Specification

## Purpose

Specview is a local-first, read-only control plane for observing agentic software development.

It does not own the work. It observes authoritative systems, normalizes their state, and projects that state into human- and agent-readable views.

The core model is:

```text
INTENT | EXECUTION | EVIDENCE | ACCEPTANCE
```

Source-control and host context connect those planes without becoming their authority.

The product must remain useful when individual optional integrations are unavailable. A GitHub failure must not hide local Git state. An Evidence failure must not hide Intent. A host index failure must not stop repository observation.

## Product thesis

Specview answers four questions:

1. What software work exists?
2. What is actively executing against it?
3. What evidence exists for the current revision?
4. Is that revision acceptable under the repository policy?

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
Federation snapshots   -> read-only remote Host observations
```

Rules:

- repository files remain authoritative for durable repo-native Intent;
- Git remains authoritative for local repository/worktree state;
- execution adapters remain authoritative for live agent execution;
- provider state degrades independently from local Git;
- Evidence is revision-scoped and never silently promoted into Acceptance;
- Acceptance is derived from policy plus Evidence for one revision;
- SQLite is disposable and rebuildable;
- federation never creates shared authority between Hosts;
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
    └── HostSnapshot
```

Relationships matter more than any one visualization. A repository may have multiple worktrees and execution sessions. A WorkItem may have multiple supporting artifacts and Evidence records. Remote Hosts remain separately authoritative even when repository instances correlate.

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

A logical execution session is not an operating-system process. One session may contain multiple helper process IDs. Process IDs are diagnostics, not durable cross-host identity.

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

The JSON host catalog remains a compatibility/history snapshot in v0.0.1. SQLite is a derived, rebuildable index used for search. Deleting the SQLite index must be safe.

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

Specview exposes the same normalized local control-plane facts through a read-only MCP stdio interface.

The MCP layer is an adapter over the same domain contracts used by the web projection. It must not create a second source of truth or gain write authority in v0.0.1.

Language-neutral fixtures protect the observable contracts so the implementation can be replaced without redefining product semantics.

## Host identity and federation

Each Host has persistent local identity. Repository instances remain Host-scoped.

Federation is conservative and read-only:

```text
Host A snapshot ─┐
Host B snapshot ─┼─> multi-host projection
Host C snapshot ─┘
```

Remote observations never override the source Host's authority.

v0.0.1 federation includes:

- persistent Host identity;
- deterministic RepositoryInstance identity;
- conservative repository correlation;
- explicit optional project identity;
- HostSnapshot v1;
- localhost-first HTTP pull transport;
- peer registry;
- credential references without persisted secret values;
- manual peer refresh;
- last-known valid remote snapshot preservation;
- freshness states: `fresh`, `stale`, `unreachable`, `never_retrieved`.

v0.0.1 does not include automatic peer discovery, background federation polling, push synchronization, remote execution, remote writes, or a shared database.

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

Host-level federation peer configuration and Host identity are stored outside repositories.

The repository-level `server` section is a compatibility artifact in v1, not a precedent for putting future Host settings in repository configuration. Its cleanup or migration requires an explicit configuration-contract change rather than an incidental removal.

## CLI surface

The product includes the local observer plus explicit read-only/control-plane utilities introduced through H01-H22.

Core commands include:

```text
specview
specview serve
specview init
specview doctor
specview version
specview help
```

Additional MCP and federation commands must preserve the same authority boundaries. Peer mutation changes only local peer configuration/cache; it does not mutate a remote Host.

## Security and privacy

Default network exposure is loopback-first.

Secrets must not be persisted in federation peer files or error text. Peer credential configuration stores references, such as environment-variable names, rather than secret values.

Host state directories and sensitive local indexes use private filesystem permissions.

Repository observation must not write product state into unrelated repositories. Specview-owned ephemeral/runtime files under `.specview/` must remain clearly separated from durable repository intent.

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

Included product capabilities are those already defined and accepted by H01-H22, plus release stabilization work that does not expand the product surface.

Before v0.0.1, development is frozen against new feature planes. Release work may:

- correct contract/documentation drift;
- fix correctness, safety, portability, performance, or packaging defects;
- improve tests and acceptance gates;
- remove accidental coupling when it can be done without destabilizing frozen public contracts.

Release work must not add a new workflow, provider, agent family, federation mode, UI paradigm, remote-write capability, or orchestration responsibility.

## Explicit non-goals for v0.0.1

- becoming a task/project management system;
- writing specification status from the UI;
- generating specifications as product authority;
- orchestrating agents;
- remote execution;
- remote repository writes;
- GitHub write operations;
- background federation daemon/polling;
- automatic peer discovery;
- shared multi-host database;
- semantic/vector search as a required dependency;
- coupling the domain model to Kanban, list, hierarchy, or graph presentation;
- requiring React, Vue, or another SPA framework.

## Definition of done

v0.0.1 is release-ready when:

1. canonical product documentation describes the architecture that is actually implemented;
2. H01-H22 accepted contracts remain green;
3. formatting, module verification, vet, race tests, build, binary smoke tests, browser semantic E2E, and release cross-build pass on the exact release head;
4. macOS and Linux release artifacts are reproducible from the release workflow;
5. installation from a GitHub Release works as a user installation, not only from a development checkout;
6. Host observation survives restart and does not require a repository-local Specview config to start;
7. Intent, Execution, Git/provider, Evidence, Acceptance, MCP, and federation failures degrade according to their documented independent authority boundaries;
8. no credential secret is persisted by federation peer state;
9. the release contains no hidden remote-write or agent-orchestration path;
10. unresolved architectural debt is explicitly classified as either release-blocking or post-v0.0.1 instead of being silently expanded into new feature work.
