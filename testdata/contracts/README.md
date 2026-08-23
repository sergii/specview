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
federation/v1-laptop.json                     HostSnapshot v1 from a laptop
federation/v1-devbox.json                     HostSnapshot v1 from a DevBox
federation/v1-projection.json                 derived two-host federation projection v1
federation/v1-aggregation-cases.json          ambiguous/conflict/transitive grouping safety cases
peers/v1.json                                 Host-level federation peer registry v1
remote-observation/v1.json                    cached remote HostSnapshot observation v1
federation-runtime/v1-status.json             multi-host freshness + aggregation projection v1
```

The current Go implementation must consume these fixtures in CI. A future Rust implementation must consume the same fixtures before it can replace the Go implementation.

Rules:

- Do not edit an existing fixture to change the meaning of a released version.
- Add a new fixture for a new compatible case.
- Introduce a new contract version or explicit migration when semantics are incompatible.
- Future public CLI JSON, MCP schemas, and federation formats must add their own versioned fixtures when they become contracts.
- Repository correlation fixtures are safety contracts: an implementation must never upgrade `ambiguous`, `distinct`, or `conflict` to `match` merely to make federation more convenient.
- Federation snapshots preserve source Host identity and observation time. A projection may correlate snapshots, but it must not erase source attribution.
- When several snapshots from one Host are supplied for a current projection, the newest `observed_at` wins. Equal timestamps with conflicting content are invalid rather than order-dependent.
- A RepositoryInstance may join a federation group only when it is a pairwise `match` with every existing member. Transitive graph connectivity is not sufficient.
- A federation `group_id` is derived projection state for an exact member set, not a durable global Repository identity.
- Peer registry files contain Host identity pins, URLs, stale thresholds, and credential references only. Secret values must never be serialized into them.
- Remote observation files preserve the source `snapshot.observed_at` unchanged. `retrieved_at` and attempt timestamps are local transport metadata.
- A failed remote retrieval may update attempt metadata but must not erase the last valid cached HostSnapshot.
- Multi-host runtime projections keep Host freshness outside immutable HostSnapshot documents. `unreachable` with a cached snapshot must preserve those source-attributed repository/session facts; `never_retrieved` must invent none.

These fixtures intentionally do not freeze HTML, CSS, Go structs, package names, SQLite schema internals, polling cadence, or a future federation UI.
