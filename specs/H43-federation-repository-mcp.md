---
specview:
  status: done
---

# H43 - Federation Repository MCP Parity

## Goal

Expose the exact H31 Host-scoped federation repository-instance drill-down through MCP over the existing H40 federation runtime projection, while centralizing exact Host/repository selection semantics so Web and MCP cannot drift into separate federation read authorities.

## Acceptance

- [x] `internal/federationruntime` owns pure exact selectors for one Host, all repository instances belonging to one Host, and one exact `(host_id, instance_id)` repository selection over an already-built `Projection`.
- [x] Selectors perform no trimming, fuzzy matching, hostname lookup, peer-name lookup, prefix matching, fallback selection, aggregation, polling, persistence, or network I/O.
- [x] H41 Web Host detail reuses the shared exact Host/repository selectors without changing visible behavior or attention linking semantics.
- [x] H31 Web federation repository detail reuses the shared exact `(host_id, instance_id)` selector without changing local-vs-remote authority behavior.
- [x] H42 `get_federation_host(host_id)` reuses the shared selector semantics and preserves its schema and output behavior.
- [x] MCP adds one read-only tool: `get_federation_repository(host_id, instance_id)`.
- [x] `host_id` and `instance_id` are required, trimmed once at the MCP input boundary, and then matched exactly against the current federation projection.
- [x] Missing, blank, or additional arguments fail as MCP invalid params before the federation reader is called.
- [x] A valid request calls the configured federation `Build` exactly once.
- [x] Projection build failures remain read-only MCP tool errors rather than JSON-RPC transport errors.
- [x] Unknown Host IDs, unknown RepositoryInstance IDs, and a valid instance paired with the wrong Host all return read-only tool errors with no fallback.
- [x] The result preserves the selected existing `federationruntime.HostStatus`, repository group ID/name/active/agents, and exact `federation.SourcedInstance`, including its sessions, worktrees, fingerprint, source repository ID, active state, and observation facts.
- [x] The result exposes the source projection generation time and its own additive result schema version without changing HostSnapshot, H20 federation projection, or `federationruntime.Projection` schema versions.
- [x] Existing pre-H43 MCP tools and argument contracts remain unchanged.
- [x] `tools/list` advertises `get_federation_repository` only when a federation reader is configured, preserving the legacy `New(...)` discovery surface.
- [x] A language-neutral MCP v6 fixture proves the contract is strictly additive over v5 by exactly one `get_federation_repository(host_id, instance_id)` tool.
- [x] Unit coverage verifies shared exact selectors, wrong-Host rejection, source/freshness preservation, exact group/instance preservation, one projection build, strict arguments, projection failure, and unknown selection behavior.
- [x] Production MCP binary smoke discovers a real local Host and RepositoryInstance from `get_federation_status`, reads the same exact pair through `get_federation_repository`, and verifies stable Host/group/instance facts plus local session/worktree arrays and repository attribution without requiring equal current-projection observation timestamps across separate stdio processes.
- [x] H43 introduces no remote writes, remote execution, polling, watcher, persistence, peer credential exposure, HostSnapshot change, federation transport change, H20 correlation change, remote repository control-plane recomputation, synthetic health score, severity ranking, or new Web capability.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, Chromium E2E, and release cross-build pass.

## Verification

- Functional head `7330786be75a2ad3af62720b66c6faf094c5db5b` passed CI #1395 (`32825113612`) across the complete pipeline.
- Formatting, module tidiness, `go vet`, and `go test -race` passed.
- Total production statement coverage: **64.7%**.
- `internal/federationruntime` statement coverage: **83.3%** after adding the shared exact selector layer.
- `internal/mcpserver` statement coverage: **81.3%** with exact repository success, strict-argument, wrong-Host, unknown-selection, projection-failure, and discovery-boundary coverage.
- The production `specview mcp` binary smoke discovered the persistent local Host and exact RepositoryInstance from `get_federation_status`, then verified `get_federation_repository` against the same stable Host/group/instance facts while treating current observation timestamps as per-projection facts.
- Existing federation CLI, HTTP, peer lifecycle, and runtime status binary smokes passed.
- Chromium semantic E2E passed after H31/H41 Web selection moved to the shared `federationruntime` selectors.
- Release archives, artifact upload, and installation command passed.
- CI #1393 stopped only at `gofmt` for the new repository projection result struct; formatting was corrected without changing behavior before the fully green #1395 run.

## Non-goals

- remote repository Intent/Evidence/Acceptance reconstruction beyond facts already present in HostSnapshot/federation projection;
- remote historical execution-session detail;
- new federation persistence or transport;
- global repository identity;
- changing H20 repository correlation semantics;
- adding a synthetic federation or repository health score.
