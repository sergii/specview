# ADR-017 - Federation Read Surface

## Status

Accepted.

## Context

H20-H23 established a deterministic, read-only multi-Host federation model: source HostSnapshots, conservative repository correlation, Host-pinned peers, freshness-aware cached observations, periodic refresh, and `specview federation status`.

That model is currently exposed primarily through the CLI. The Host web UI and MCP server still present only local-Host facts, even though the federation projection is already a stable read model.

## Decision

Expose the existing `federationruntime.Projection` as a first-class read surface without creating a second federation model.

H28 adds:

- a read-only Web federation page built from the existing projection builder;
- a read-only MCP `get_federation_status` tool returning the same projection;
- language-neutral MCP tool-contract coverage for the additive tool;
- browser and built-binary coverage for the new surfaces.

The projection builder remains authoritative for derived federation status. Web and MCP are adapters only.

## Semantics

- local Host facts are freshly built when the projection is read;
- cached remote snapshots retain source Host attribution;
- `fresh`, `stale`, and `unreachable` peers with cached snapshots remain visible;
- `never_retrieved` peers remain visible without invented repository facts;
- repository groups remain derived correlation state, not durable global identity;
- transport timestamps and freshness metadata do not rewrite source HostSnapshot facts.

## MCP compatibility

The MCP protocol version remains unchanged. `get_federation_status` is an additive read-only tool with no arguments. Existing tool names, arguments, and result contracts are unchanged.

## Non-goals

- automatic peer discovery;
- push synchronization;
- remote execution or writes;
- changing HostSnapshot v1;
- changing correlation semantics;
- exposing peer secrets;
- making the browser or MCP server a federation authority.
