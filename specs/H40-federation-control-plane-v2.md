---
specview:
  status: done
---

# H40 - Federation Control Plane v2

## Goal

Carry the H38/H39 read-only Host control-plane projection between Specview Hosts without breaking the frozen H20/H21 HostSnapshot v1 transport, so a federation view can show per-Host Intent, logical Execution, native Evidence, Acceptance, and factual attention while preserving source-host authority.

## Acceptance

- [x] H20/H21 HostSnapshot v1 remains strictly decodable and can still be served at `GET /v1/federation/snapshot` without a `control_plane` field.
- [x] HostSnapshot v2 adds one optional-on-the-Go-model but required-for-v2 `control_plane` projection using the existing `controlplane.GetHostControlPlaneResult` contract.
- [x] HostSnapshot v2 validation requires a valid shared control-plane schema and matching hostname authority.
- [x] The local federation snapshot builder emits v2 and reuses `controlplane.Reader.GetHostControlPlane`; it does not duplicate Host aggregation semantics.
- [x] The HTTP server serves v1 and v2 simultaneously from one local source, down-projecting v2 to the frozen v1 wire shape for the v1 endpoint.
- [x] New federation clients prefer `/v2/federation/snapshot` and fall back to `/v1/federation/snapshot` when the peer does not expose v2.
- [x] Existing explicit v1 and v2 peer URLs remain valid; remote cleartext HTTP security, redirects, response bounds, credentials, and Host ID pinning remain unchanged.
- [x] A new client can read an old v1 peer, and an old v1 client can read a new H40 server.
- [x] Remote observation persistence can store either v1 or v2 snapshots without changing the observation-store version.
- [x] Federation repository correlation and the nested H20 repository projection remain unchanged.
- [x] The outer federation runtime projection is versioned for the new per-Host control-plane field.
- [x] Local and v2 peer Host rows expose the source Host's control plane; v1 peers remain explicitly `control plane unavailable` rather than being interpreted as zero/healthy.
- [x] The Federation Web page shows concise per-Host control-plane facts without adding a synthetic cross-Host health score.
- [x] MCP `get_federation_status` returns the same per-Host control-plane facts through the existing federation projection.
- [x] H40 introduces no remote writes, global health score, severity ranking, distributed database, new repository correlation identity, Evidence schema, Acceptance contract, execution-history contract, repository config contract, or Host catalog contract change.
- [x] Unit/contract tests cover v1 decode, v2 decode/validation, v2-to-v1 down-projection, client v2 preference with v1 fallback, and runtime Host control-plane attribution.
- [x] Chromium E2E covers Federation Host control-plane visibility while preserving freshness/source attribution.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional H40 head `877b38cbac01f96a08dae21b07086c7406d30773` passed CI #1333 (run `32816979820`) end to end:

- formatting, modules, vet, and `go test -race` passed;
- total production statement coverage: **64.1%**;
- `internal/federation`: **81.7%**;
- `internal/federationhttp`: **77.5%**;
- `internal/federationruntime`: **81.7%**;
- `internal/mcpserver`: **77.2%**;
- `internal/web`: **50.9%**;
- production MCP binary smoke confirms the local `get_federation_status.hosts[].control_plane` is structurally identical to `get_host_control_plane` while the nested H20 repository projection remains schema v1;
- production federation CLI smoke confirms `federation snapshot` emits v2 with the shared Host control plane and frozen v1 fixtures still aggregate unchanged;
- production federation HTTP smoke confirms `/v1` remains control-plane-free, `/v2` carries the Host control plane, and a new client prefers v2 while falling back to v1;
- production federation runtime smoke confirms cached v2 control-plane facts survive an unreachable peer and never-retrieved peers do not invent them;
- Chromium semantic tests confirm local and cached-unreachable Host control-plane visibility while preserving source/freshness attribution;
- release archives and installation command passed.

CI #1323 stopped at gofmt before compilation. CI #1329 reached race tests and exposed two contract-test details: empty `attention: []` needed lossless clone preservation, and one MCP assertion depended on JSON whitespace. Both were corrected before the fully green functional run.

## Non-goals

- merging Host control planes into one synthetic federation health result;
- linking remote attention rows directly to mutable remote repository state;
- changing H20 repository correlation semantics;
- removing the v1 endpoint;
- remote execution or mutation;
- WebSocket/push federation.
