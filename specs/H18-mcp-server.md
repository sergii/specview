---
specview:
  status: in_progress
---

# H18 - MCP Server

## Goal

Expose Specview's normalized, read-only control-plane facts to coding agents through MCP without creating a second state model or embedding an LLM.

The intended boundary is:

```text
normalized Specview core
        ↓
controlplane.Reader
        ↓
MCP adapter / CLI JSON / future APIs
```

MCP is a transport and tool adapter. It does not own repository, execution, Git, forge, Evidence, or Acceptance semantics.

## First transport

H18 starts with local stdio JSON-RPC:

```text
specview mcp
```

The transport must keep stdout protocol-pure: no banners, progress messages, or runtime logs may be written to stdout.

## Initial read-only tools

```text
list_repositories
get_repository
list_active_sessions
list_worktrees
```

The next H18 sub-slice extends the same control-plane contract with:

```text
get_work_item
get_evidence
get_acceptance
```

Every MCP tool is read-only, non-destructive, idempotent, and closed-world with respect to Specview's observed state.

## Compatibility strategy

The control-plane JSON models are implementation-neutral and versioned independently from Go implementation details. MCP structured content uses those models directly.

The initial stdio transport supports the stable MCP 2025-11-25 protocol generation. A later transport/SDK upgrade must not rename tools or silently change normalized result semantics.

Language-neutral contract fixtures and black-box protocol tests are the compatibility boundary for a future Go -> Rust implementation replacement.

## Acceptance criteria

- [x] control-plane read model exists independently from MCP transport;
- [x] repositories combine persisted catalog history with live execution observations;
- [x] repository details expose degradable Git and forge context;
- [x] active sessions expose normalized agent/session/worktree facts;
- [x] worktrees expose revision, branch, dirty, upstream, ahead/behind facts;
- [x] MCP stdio transport exposes the initial four read-only tools;
- [x] `specview mcp` is wired into the production CLI;
- [x] CI exercises MCP through the built binary rather than only Go handlers;
- [ ] WorkItem facts are available through the control plane and MCP;
- [ ] Evidence facts are available through the control plane and MCP;
- [ ] Acceptance decisions are available through the control plane and MCP;
- [ ] language-neutral MCP fixtures cover the completed H18 public contract;
- [ ] final gofmt, module, vet, race, coverage, browser, MCP binary, and release gates pass.

## Out of scope

- write tools;
- running tests or CI;
- mutating Git or forge state;
- HTTP MCP transport;
- OAuth;
- multi-host federation;
- A2A;
- embedded LLM reasoning.
