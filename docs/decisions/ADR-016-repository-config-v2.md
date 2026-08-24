# ADR-016 - Repository Config v2 Removes Host Networking

## Status

Accepted.

## Context

The original `.specview.yaml` v1 contract contains a repository-scoped `server` section with `host` and `port`. The Host dashboard no longer consumes those fields: the local observer owns its loopback listener, while federation peer/runtime state is Host-scoped and stored outside repositories.

Keeping networking fields in newly generated repository Intent config would imply an ownership relationship that the product model no longer has.

## Decision

Introduce repository configuration version 2.

Version 2 contains repository identity, Intent adapter settings, and Acceptance policy. It does not contain Host networking fields.

`specview init` generates version 2 and never writes a `server` section.

The v1 reader remains supported and strict. Existing v1 files with valid `server.host` and `server.port` continue to load and are not rewritten by `specview init`.

A v2 file that contains a `server` section is rejected. Host networking must not silently leak back into repository configuration.

The Host dashboard listener remains the existing loopback-only `127.0.0.1:7331` behavior in this slice. H27 does not add bind flags or a new Host settings file.

## Compatibility

- v1 repository config: readable, unchanged on disk.
- v2 repository config: canonical writer format.
- unknown versions: rejected.
- v2 plus `server`: rejected.
- MCP, federation, Evidence and Acceptance wire contracts: unchanged.

## Consequences

Repository Intent configuration no longer claims ownership of Host networking. Existing repositories remain readable without migration tooling, while all newly initialized repositories use the cleaner contract.
