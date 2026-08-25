---
specview:
  status: in_progress
---

# H44 - Federation Repository Control Plane v3

## Goal

Carry the existing H36/H37 read-only repository control-plane summary from its source Host inside federation snapshots so exact local and cached remote repository instances expose Intent, logical Execution, native Evidence, and Acceptance without recomputing remote repository state on the observing Host.

## Acceptance

- [ ] HostSnapshot v1 and v2 remain decodable and keep their existing wire semantics.
- [ ] HostSnapshot v3 adds one source-attributed `control_plane` field to each RepositoryInstance using the existing `controlplane.GetRepositoryControlPlaneResult` contract.
- [ ] The local snapshot builder emits v3 and obtains each repository control plane through the existing source-Host `GetRepositoryControlPlane(repository_id)` authority rather than reimplementing H36 composition.
- [ ] Repository control-plane validation requires the shared control-plane schema, source repository ID, repository name, and source Host name to match the containing RepositoryInstance and HostSnapshot.
- [ ] A v3 snapshot requires repository control-plane facts for every RepositoryInstance; failure to obtain them fails snapshot construction rather than silently inventing zeros.
- [ ] `V2()` down-projects v3 by removing repository control-plane fields while retaining the H40 Host control plane; `V1()` removes both Host and repository control-plane fields.
- [ ] `/v3/federation/snapshot` serves the current v3 snapshot while `/v2` and `/v1` remain available as exact down-projections.
- [ ] Federation clients prefer v3, then fall back to v2 and v1 when newer endpoints are unavailable; explicit v1/v2/v3 peer URLs remain valid.
- [ ] Existing transport security, redirect refusal, response bounds, Host ID pinning, and observation-store version remain unchanged.
- [ ] The H20 repository correlation algorithm and `federation.Projection` schema version remain unchanged; repository control-plane data is carried as source-attributed instance data and does not affect grouping.
- [ ] The outer `federationruntime.Projection` schema version advances to declare the new repository-instance facts.
- [ ] Local v3 and cached v3 peer instances preserve their source repository control plane through aggregation and runtime projection; v1/v2 peers expose it as unavailable rather than zero or healthy.
- [ ] `/federation/repository?host=...&instance=...` displays the captured repository control-plane facets when present and never recomputes remote Intent, Evidence, Acceptance, or Execution.
- [ ] Stale or unreachable cached v3 repository instances retain their last valid repository control-plane facts alongside existing freshness and transport-error attribution.
- [ ] H43 `get_federation_repository(host_id, instance_id)` returns the same captured repository control plane through its existing exact `instance` result without adding another MCP tool or changing MCP arguments.
- [ ] `get_federation_status` likewise carries the same source-attributed repository control-plane facts through the runtime projection.
- [ ] Unit and compatibility coverage prove v1/v2 decode, v3 validation, v3-to-v2/v1 down-projection, exact builder attribution, client v3 preference/fallback, observation persistence, aggregation preservation, and runtime local/cached attribution.
- [ ] Chromium E2E covers repository control-plane visibility for local and cached remote repository instances, including unavailable older-peer semantics and stale/unreachable attribution.
- [ ] Production federation CLI/HTTP/runtime and MCP binary smokes verify v3 while preserving v1/v2 compatibility.
- [ ] H44 introduces no remote writes, remote execution, on-demand remote fetching, observing-Host recomputation, global health score, severity ranking, correlation change, observation-store migration, execution-history transport, per-work-item remote detail, Evidence record ledger transport, or push/WebSocket federation.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, Chromium E2E, and release cross-build pass.

## Non-goals

- remote mutable repository navigation;
- remote WorkItem/Evidence/Acceptance detail endpoints;
- federation-wide readiness or health synthesis;
- changing repository identity or correlation semantics;
- changing local H36/H37 repository control-plane semantics;
- replacing pull/cached federation with push transport.
