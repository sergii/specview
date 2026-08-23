---
specview:
  status: done
---

# H18 - MCP Server

## Goal

Expose Specview's normalized, read-only control-plane facts to coding agents through MCP without creating a second state model or embedding an LLM.

The implemented boundary is:

```text
normalized Specview domains
        ↓
projectstate
        ↓
controlplane.Reader
        ↓
MCP adapter / future CLI JSON / future APIs
```

`projectstate` owns reusable repository-level resolution for Intent, Evidence, revision identity, and Acceptance. Web and MCP consume the same semantics rather than maintaining parallel implementations.

`controlplane` defines implementation-neutral, versioned JSON read contracts. MCP is a transport/tool adapter over those contracts. It does not own repository, execution, Git, forge, Evidence, or Acceptance semantics.

## Transport

H18 ships local stdio JSON-RPC:

```text
specview mcp
```

Stdout is protocol-pure: no startup banner, progress text, or runtime logging is written to the MCP stream.

The initial transport targets MCP protocol version `2025-11-25` and accepts standard client `capabilities`, `clientInfo`, and `_meta` fields. Specview-specific tool arguments remain strictly validated.

## Read-only tool surface

H18 exposes eight deterministic tools:

```text
list_repositories
get_repository
list_active_sessions
list_worktrees
list_work_items
get_work_item
get_evidence
get_acceptance
```

The intended agent discovery flow is:

```text
list_repositories
        ↓
list_work_items
        ↓
get_work_item
   ├── get_evidence
   └── get_acceptance
```

`list_work_items` exposes lightweight normalized WorkItem summaries and stable `work_item_id` values. `get_work_item` returns the full normalized Intent artifact, including body and relationships.

Every MCP tool is annotated read-only, non-destructive, idempotent, and closed-world with respect to Specview's observed state.

## Repository and execution facts

`list_repositories` combines persisted host catalog history with current execution-adapter observations. A repository that has only just become active can therefore appear before it has become durable history.

`get_repository` projects:

- repository identity and path;
- active agents;
- local Git remote and worktrees;
- branch, revision, dirty state, upstream, ahead/behind facts;
- independently degradable forge context and pull requests.

`list_active_sessions` exposes normalized coding-agent sessions rather than raw process trees.

`list_worktrees` exposes repository-local Git worktree facts without turning MCP into a Git mutation interface.

## WorkItem, Evidence, and Acceptance facts

WorkItem discovery and detail are resolved through the repository's configured or detected Intent adapter.

Evidence is loaded through the normalized native Evidence adapter and remains revision-scoped. Records are returned newest observation first.

Acceptance uses the same shared `projectstate` and fail-closed revision semantics as the Web projection:

- clean worktree with known HEAD may evaluate `git:<head>`;
- dirty or unresolved worktree cannot inherit Acceptance from clean HEAD Evidence;
- unconfigured policy returns `unconfigured` without requiring Git inspection;
- configured policy evaluates normalized logical checks against exact-revision Evidence.

## Compatibility strategy

The control-plane JSON models are implementation-neutral and versioned independently from Go implementation details. MCP `structuredContent` uses those models directly and also mirrors the result as JSON text content for client compatibility.

Language-neutral fixtures freeze the public H18 contracts for:

```text
MCP tool names + arguments
list_repositories
list_work_items
get_work_item
get_evidence
get_acceptance
```

These fixtures, protocol transcript tests, and built-binary black-box tests form the compatibility boundary for a future Go -> Rust implementation replacement.

A later MCP transport or SDK upgrade must not silently rename tools or change normalized result semantics. HTTP/OAuth or a newer MCP protocol generation can be introduced behind the same control-plane contract.

## Verification

The production CI gate now verifies:

```text
bash syntax
Go formatting
module hygiene
go vet
go test -race
production coverage floor
Go build
built-binary MCP stdio flow
Playwright Chromium semantic E2E
Linux/macOS release archives
```

The built-binary MCP test creates a real temporary Git repository with:

```text
.specview.yaml
WorkItem H18
clean Git revision
revision-scoped Evidence
host catalog entry
```

It then communicates with the compiled `specview mcp` process over stdin/stdout and verifies repository discovery, WorkItem discovery/detail, Evidence, and an `accepted` Acceptance decision for the exact revision.

The completed H18 production coverage baseline is 64.5%. At the H18 gate:

- `controlplane`: 76.4%
- `mcpserver`: 75.2%
- `projectstate`: 67.5%
- `acceptance`: 92.2%
- `revision`: 96.7%

## Acceptance criteria

- [x] control-plane read model exists independently from MCP transport;
- [x] reusable project-state semantics are shared by Web and MCP;
- [x] repositories combine persisted catalog history with live execution observations;
- [x] repository details expose degradable Git and forge context;
- [x] active sessions expose normalized agent/session/worktree facts;
- [x] worktrees expose revision, branch, dirty, upstream, ahead/behind facts;
- [x] MCP stdio transport is wired into the production `specview mcp` CLI;
- [x] MCP accepts standard initialization metadata while keeping Specview tool arguments strict;
- [x] all MCP tools are explicitly read-only and non-destructive;
- [x] WorkItems are discoverable with `list_work_items`;
- [x] full WorkItem Intent facts are available through `get_work_item`;
- [x] revision-scoped Evidence is available through `get_evidence`;
- [x] deterministic Acceptance is available through `get_acceptance`;
- [x] dirty worktrees cannot inherit Acceptance from clean HEAD Evidence;
- [x] language-neutral MCP fixtures cover the completed H18 public structured-content contract;
- [x] CI exercises the full WorkItem -> Evidence -> Acceptance path through the built binary;
- [x] gofmt, module, vet, race, coverage, browser, MCP binary, and release gates pass on the completed functional slice.

## Out of scope

- write tools;
- running tests or CI from MCP;
- mutating Git or forge state;
- HTTP MCP transport;
- OAuth;
- MCP resources;
- multi-host federation;
- A2A;
- embedded LLM reasoning.
