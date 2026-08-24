---
specview:
  status: in_progress
---

# H27 - Repository Config v2

## Goal

Remove legacy Host networking ownership from newly generated repository configuration while preserving strict v1 compatibility.

## Acceptance

- [ ] `specview init` generates `.specview.yaml` version 2.
- [ ] Generated v2 config contains no `server` section.
- [ ] Version 2 loads repository identity, Intent adapter settings, and Acceptance policy.
- [ ] Existing version 1 config with valid `server.host` and `server.port` still loads.
- [ ] `specview init` does not rewrite an existing valid v1 config.
- [ ] Version 2 with a `server` section fails closed.
- [ ] Unknown config versions fail closed.
- [ ] Specview's own repository uses v2 as dogfood.
- [ ] Language-neutral v1 and v2 config fixtures are both covered.
- [ ] Dashboard bind behavior remains loopback `127.0.0.1:7331`.
- [ ] MCP, federation, Evidence, Acceptance and HostSnapshot contracts are unchanged.
- [ ] Full CI, race tests, browser semantic E2E and release cross-build pass.

## Non-goals

- configurable Host bind addresses;
- a general Host settings file;
- rewriting user repositories automatically;
- removing the v1 reader;
- changing federation peer configuration.
