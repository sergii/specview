# ADR-008: Give hosts stable identity and correlate repositories conservatively

- Status: Accepted
- Date: 2026-08-23

## Context

Specview is local-first, but the same logical repository may be active on several machines at once. A laptop and a devbox can each have their own checkout, worktrees, agent sessions, Git state, and Evidence while still belonging to one logical Repository.

Hostnames and filesystem paths are useful labels and locators, but neither is durable identity. Hostnames can change. Paths differ across machines and may change on one machine. Forge identity is also optional because a valid repository may have Git without a forge, or no Git at all.

ADR-006 already separates logical `Repository` from host-local `RepositoryInstance`. H19 defines the identity contract needed before any synchronization transport is built.

## Decision

### Stable Host identity

Each Specview installation creates one persistent opaque Host ID in a separate versioned host identity file.

```text
host:<uuid>
```

The Host ID is generated once and survives process restarts and hostname changes. The current OS hostname is a mutable display label, not identity.

The existing `catalog.json` v1 format remains unchanged. Host identity is persisted separately so H19 does not silently migrate the catalog contract.

### Repository and RepositoryInstance

A logical Repository is not identified by a filesystem path.

A `RepositoryInstance` is one realization of a Repository on one Host. Its host-local identity is derived from:

```text
Host ID + canonical local repository root
```

Changing the local path may produce a new RepositoryInstance identity while correlation can still attach the new instance to the same logical Repository.

### Explicit project identity

`.specview.yaml` may optionally declare a stable project identity:

```yaml
project:
  id: specview:sergii/specview
  name: Specview
  root: .
```

`project.id` is an opaque, user-controlled logical identity hint. It has the highest correlation priority, but contradictory strong evidence must surface a conflict rather than being silently ignored.

### Correlation evidence

Each RepositoryInstance can advertise a normalized fingerprint:

```text
explicit Specview project ID
normalized repository name
normalized Git remote
forge provider + forge repository identity
```

Correlation is conservative and returns one of four outcomes:

```text
match
ambiguous
distinct
conflict
```

Rules:

1. matching explicit project IDs are the strongest positive signal;
2. different explicit project IDs mean `distinct`;
3. contradictory complete forge identities or contradictory normalized Git remotes mean `distinct`, unless the same explicit project ID claims both belong together, in which case the result is `conflict`;
4. without explicit IDs, normalized repository names must match before Git/forge facts can corroborate a `match`;
5. same normalized name plus matching Git remote or matching forge identity is a `match`;
6. same normalized name without corroborating Git/forge facts is `ambiguous`, not an automatic merge;
7. different normalized names without a shared explicit project ID are `distinct` in the initial model;
8. missing Git or forge data never makes a repository invalid.

A federation layer must never silently merge `ambiguous` or `conflict` instances.

### Normalization

Repository names are compared case-insensitively after trimming whitespace, converting backslashes to `/`, removing duplicate/trailing separators, and removing a trailing `.git` suffix.

Git remotes normalize common SSH and HTTPS forms so these identify the same remote:

```text
git@github.com:sergii/specview.git
https://github.com/sergii/specview.git
ssh://git@github.com/sergii/specview.git
```

The normalized comparison value is an implementation-neutral identity hint, not a URL to display to the user.

### Federation authority

Federation does not create a new global source of truth for host-local facts.

Each Host remains authoritative for facts it directly observes or owns:

```text
Host
  RepositoryInstance filesystem root
  local Git/worktree state
  local execution sessions/processes
  host-local activity timestamps
  locally collected Evidence
```

A federation consumer may cache and project those facts, but cached remote state is always attributed to its source Host and observation time. It must not be rewritten as if it were locally observed.

Logical Repository grouping is derived correlation state. It does not mutate the source RepositoryInstance identities. If correlation later changes from `match` to `conflict`, the original host-local facts remain intact and can be regrouped without data loss.

Remote facts may become stale or unavailable. Staleness affects projection freshness, not historical identity. A disconnected Host is not interpreted as having zero active work unless the source Host explicitly reported that state.

This extends ADR-001 authority-by-fact-type into the multi-host topology: federation aggregates authority, it does not replace it.

## Consequences

- laptop and devbox can have stable identities independent of hostname;
- `RepositoryInstance` can be addressed without pretending a local path is global identity;
- federation can combine instances only when the identity contract says it is safe;
- local-only repositories remain first-class;
- explicit identity can resolve otherwise ambiguous same-name repositories;
- accidental reuse of an explicit identity can be detected as a conflict;
- stale remote snapshots cannot silently erase source-host state;
- correlation can be recomputed without mutating source facts;
- H20 can implement transport/synchronization without redefining identity semantics;
- future Rust code can be checked against the same language-neutral correlation fixtures.

## Persistence

Host identity is versioned separately from the host repository catalog. Existing catalog v1 files remain readable without migration.

The host identity file contains only durable identity facts. Runtime labels such as current hostname are observed separately.

## Out of scope

- network discovery;
- synchronization transport;
- conflict-resolution UI;
- automatic mutation of `.specview.yaml`;
- globally unique logical Repository IDs for repositories that remain ambiguous;
- graph database storage;
- multiple forge mirrors for one normalized Repository.
