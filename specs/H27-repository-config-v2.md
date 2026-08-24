---
specview:
  status: done
---

# H27 - Repository Config v2

## Goal

Remove legacy Host networking ownership from newly generated repository configuration while preserving strict v1 compatibility.

## Acceptance

- [x] `specview init` generates `.specview.yaml` version 2.
- [x] Generated v2 config contains no `server` section.
- [x] Version 2 loads repository identity, Intent adapter settings, and Acceptance policy.
- [x] Existing version 1 config with valid `server.host` and `server.port` still loads.
- [x] `specview init` does not rewrite an existing valid v1 config.
- [x] Version 2 with a `server` section fails closed.
- [x] Unknown config versions fail closed.
- [x] Specview's own repository uses v2 as dogfood.
- [x] Language-neutral v1 and v2 config fixtures are both covered.
- [x] Dashboard bind behavior remains loopback `127.0.0.1:7331`.
- [x] MCP, federation, Evidence, Acceptance and HostSnapshot contracts are unchanged.
- [x] Full CI, race tests, browser semantic E2E and release cross-build pass.

## Verification

Functional head `622d05489fb450ee36fc9c4ce0924334e0338778` passed CI #1099 (`32676468692`):

- formatting/modules/vet/race ✅
- total production coverage: **65.0%** ✅
- `internal/config`: **81.2%** ✅
- build ✅
- MCP binary stdio smoke ✅
- federation CLI/HTTP/peer/runtime binary smokes ✅
- Chromium semantic E2E ✅
- Linux/macOS amd64/arm64 release archive cross-build ✅

README and canonical `SPEC.md` now describe repository config v2 as the writer contract and v1 as a compatibility reader only.

## Non-goals

- configurable Host bind addresses;
- a general Host settings file;
- rewriting user repositories automatically;
- removing the v1 reader;
- changing federation peer configuration.
