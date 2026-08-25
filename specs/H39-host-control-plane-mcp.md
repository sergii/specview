---
specview:
  status: done
---

# H39 - Host Control Plane MCP Parity

## Goal

Expose the H38 local Host control-plane projection through MCP without creating a second aggregation authority, so Web and agent clients observe the same deterministic Intent, logical Execution, native Evidence, Acceptance, and attention semantics.

## Acceptance

- [x] Host control-plane aggregation lives in `internal/controlplane` and is reused by the H38 Web Host projection.
- [x] The shared projection preserves H38 Intent, logical Execution, native Evidence, Acceptance, and deterministic attention semantics.
- [x] The shared projection preserves H38 behavior that the Host summary remains global while repository search filters only repository Results.
- [x] `controlplane.Reader` exposes a read-only `GetHostControlPlane` result with schema version and Host identity.
- [x] MCP `tools/list` advertises `get_host_control_plane` only when the configured reader supports the Host control-plane reader contract.
- [x] `get_host_control_plane` accepts no arguments and returns structured content from the shared control-plane projection.
- [x] The MCP result exposes Host Intent counts, logical Execution counts/latest session, native Evidence counts/latest record, Acceptance counts, and factual attention rows.
- [x] H39 introduces no synthetic Host health score, severity ranking, write action, persistence, watcher, polling loop, network authority, Evidence schema, Acceptance contract, execution-history contract, repository config contract, Host catalog contract, SQLite model, federation wire model, or source-control contract change.
- [x] Existing repository `get_repository_control_plane`, execution-history, Evidence, Acceptance, and federation MCP tools remain unchanged.
- [x] Unit coverage verifies tool discovery and structured Host projection output.
- [x] Production MCP binary smoke verifies `get_host_control_plane` against a real catalog, repository config, Git revision, Evidence record, and Acceptance evaluation.
- [x] Existing H38 Host Web unit and Chromium tests remain green after moving aggregation into the shared control-plane package.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

- Functional CI #1299 passed the full pipeline after the shared projection move and MCP addition.
- `go test -race` passed across `./cmd/... ./internal/...`.
- Total production statement coverage: 63.9%.
- `internal/controlplane` statement coverage: 54.1% after absorbing the Host aggregation implementation.
- `internal/mcpserver` statement coverage: 77.2%.
- Production `specview mcp` stdio smoke exercised `get_host_control_plane` with a real catalog, repository config, Git revision, native Evidence, and Acceptance.
- Chromium semantic tests passed with the H38 Host Web projection reading the shared `internal/controlplane` result.
- Release archives and installation command passed.
- CI #1297 stopped only at `gofmt` for the newly moved Go file; formatting was corrected without changing projection semantics before #1299.

## Non-goals

- aggregating remote federation Hosts into the local Host control plane;
- adding Host write operations;
- exposing a global green/red status;
- adding query/filter arguments to `get_host_control_plane`;
- changing the H38 UI layout or attention ordering.
