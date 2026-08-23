# Specview contract fixtures

Files in this directory are language-neutral conformance fixtures for versioned Specview formats.

They are product contracts, not Go-specific test helpers.

Current fixtures:

```text
config/v1-acceptance.yaml                     .specview.yaml version 1 Acceptance policy
config/v1-project-id.yaml                     .specview.yaml version 1 explicit project identity
evidence/v1-passed.json                       native Evidence version 1
catalog/v1.json                               host catalog persistence version 1
host/v1.json                                  persistent Host identity version 1
repository-identity/v1-correlation.json       RepositoryInstance and cross-host correlation contract
mcp/v1-tools.json                             MCP tool surface contract
mcp/v1-list-repositories.json                 MCP repository list structured content
mcp/v1-list-work-items.json                   MCP WorkItem discovery structured content
mcp/v1-get-work-item.json                     MCP WorkItem detail structured content
mcp/v1-get-evidence.json                      MCP Evidence structured content
mcp/v1-get-acceptance.json                    MCP Acceptance structured content
```

The current Go implementation must consume these fixtures in CI. A future Rust implementation must consume the same fixtures before it can replace the Go implementation.

Rules:

- Do not edit an existing fixture to change the meaning of a released version.
- Add a new fixture for a new compatible case.
- Introduce a new contract version or explicit migration when semantics are incompatible.
- Future public CLI JSON, MCP schemas, and federation formats must add their own versioned fixtures when they become contracts.
- Repository correlation fixtures are safety contracts: an implementation must never upgrade `ambiguous`, `distinct`, or `conflict` to `match` merely to make federation more convenient.

These fixtures intentionally do not freeze HTML, CSS, Go structs, package names, SQLite schema internals, or a future federation transport that has not yet been designed.
