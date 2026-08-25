---
specview:
  status: done
---

# H42 - Federation Host MCP Parity

## Goal

Expose the exact H41 federation Host drill-down through MCP as a strictly additive read-only tool over the existing H40 federation runtime projection, so agent clients can inspect one source-attributed local or remote Host without downloading and filtering the complete multi-Host projection themselves or creating a second aggregation authority.

## Acceptance

- [x] MCP adds one read-only tool: `get_federation_host(host_id)`.
- [x] `host_id` is required, trimmed once at the input boundary, and then matched exactly against the current `federationruntime.Projection`; no prefix, fuzzy, hostname, peer-name, or fallback matching is introduced.
- [x] Invalid, missing, blank, or additional arguments fail as MCP invalid params before the federation reader is called.
- [x] The tool calls the configured federation `Build` exactly once for a valid request.
- [x] Projection build failures remain read-only tool errors rather than JSON-RPC transport errors.
- [x] An unknown exact Host ID returns a read-only tool error and does not fall back to another Host.
- [x] The result preserves the selected existing `federationruntime.HostStatus`, including local/peer source attribution, freshness, snapshot availability, retrieval metadata, source age, last error, and H40 Host control-plane facts when available.
- [x] The result includes only repository instances belonging to the selected Host, derived from the existing nested H20 repository projection in its current deterministic order.
- [x] Repository rows preserve their existing group ID, group name, active flag, agents, and exact `federation.SourcedInstance`; H42 performs no new correlation.
- [x] The result exposes the source projection generation time and its own additive result schema version without changing `federationruntime.ProjectionSchemaVersion`, H20 projection schema, or HostSnapshot schema.
- [x] Existing `get_federation_status` behavior and all pre-H42 MCP tool names/arguments remain unchanged.
- [x] `tools/list` advertises `get_federation_host` only when the Server has a configured federation reader, preserving the legacy `New(...)` discovery surface while the production configured MCP server exposes H42.
- [x] A language-neutral MCP v5 fixture proves the contract is strictly additive over v4 by exactly one `get_federation_host(host_id)` tool.
- [x] Unit coverage verifies exact remote Host selection, source/freshness/control-plane preservation, Host-only repository filtering, one projection build, strict argument validation, projection failure behavior, and unknown Host behavior.
- [x] Production MCP binary smoke discovers the real local Host ID from `get_federation_status`, reads that exact Host through `get_federation_host`, and verifies parity of stable Host/control-plane facts plus local repository attribution.
- [x] H42 introduces no remote writes, execution, polling, watcher, persistence, peer credential exposure, HostSnapshot change, federation transport change, H20 correlation change, Host control-plane recomputation, synthetic health score, severity ranking, or Web behavior change.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, Chromium E2E, and release cross-build pass.

## Verification

Functional implementation head `aa01dae63afcbc424a0fbc5ae75a708bdc04126c` passed CI #1372 (`32821654648`) fully green: formatting, module verification, vet, race tests, coverage gate, build, production MCP stdio smoke, federation CLI/HTTP/peer/runtime smokes, Chromium semantic tests, release archives, artifact upload, and installation command.

Production statement coverage on that head is 64.6% total. `internal/mcpserver` is 80.7%; existing federation layers remain `internal/federation` 81.7%, `internal/federationhttp` 77.5%, and `internal/federationruntime` 81.7%.

The production MCP smoke discovers the persistent local Host ID from `get_federation_status` before requesting `get_federation_host`. Separate stdio processes create fresh current projections, so local `observed_at` timestamps are intentionally allowed to differ while exact Host identity, source, hostname, snapshot availability, control-plane facts, and repository attribution must match.

## Non-goals

- remote execution or writes;
- listing historical remote execution sessions beyond H40 facts;
- adding repository-level remote MCP drill-down in this slice;
- changing federation peer discovery or freshness rules;
- changing the H41 Web Host detail;
- adding a synthetic federation or Host health score.
