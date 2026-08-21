---
specview:
  status: in_progress
---

# H16 - SQLite Host Index and Search

## Goal

Add a rebuildable local SQLite projection for host-level repository discovery, history lookup, and search without changing authority boundaries.

```text
Filesystem / Git / Execution / Provider
                 ↓
            Host Catalog
                 ↓
         SQLite Host Index
                 ↓
          query / search
```

SQLite is not a repository source of truth and does not own Git, specifications, execution state, or provider state.

## Authority

The existing authority model remains:

- filesystem and Git describe repository state and durable intent;
- execution adapters describe running execution;
- provider adapters describe remote forge context;
- the host catalog keeps the current compatibility/history snapshot;
- SQLite is a derived host index that can be deleted and rebuilt.

H16 deliberately does not make repository files depend on SQLite and does not write anything inside observed repositories.

## Storage

Default location is beside the existing host catalog:

```text
macOS
~/Library/Application Support/Specview/
├── catalog.json
└── index.sqlite

Linux
~/.local/state/specview/
├── catalog.json
└── index.sqlite
```

`XDG_STATE_HOME` remains respected through the catalog state directory.

The SQLite schema is versioned and starts with:

```text
meta
repositories
sessions
```

Repository rows project identity, path, first/last observation, and specification convention label.

Session rows project historical execution identity, agent, PID, start/end timestamps, and active state.

## Rebuild semantics

The index is synchronized from the in-memory host catalog.

A full snapshot replacement is acceptable at the current host scale because expected cardinality is tens or hundreds of repositories and sessions, not millions.

To avoid writes every polling interval, the index computes a structural fingerprint that includes:

- repository identity/name/root/convention;
- session identity/agent/PID/start/end/active state.

`last_seen` heartbeat changes alone do not cause SQLite rewrites.

Deleting `index.sqlite` must be safe. Specview recreates schema and projects the current catalog again at startup.

## SQLite driver

Use the CGO-free `modernc.org/sqlite` driver.

H16 pins `v1.38.2` because it retains Go 1.23 compatibility and supports the existing release targets:

- darwin/amd64;
- darwin/arm64;
- linux/amd64;
- linux/arm64.

The storage layer itself is not Darwin/Linux-specific. Windows support remains blocked only by execution process discovery, not by this index design.

## Search

The host page gains a minimal repository search input.

Search is backed by SQLite and matches case-insensitively against:

- repository name;
- repository root/path;
- specification convention label;
- observed execution agent names, including historical sessions.

Search returns repository IDs only. The current in-memory catalog remains responsible for rendering live repository state, ordering, and links.

This separation prevents stale index rows from becoming UI authority.

## Failure semantics

SQLite is derived and optional for core observation.

If the index cannot be opened or synchronized:

- Specview logs a warning;
- host observation continues;
- repository pages continue to work;
- search reports that the host index is unavailable.

Index failure must never stop Codex discovery or make an observed repository disappear from the unfiltered host page.

## Acceptance criteria

- [x] SQLite host index is a separate package from portable host domain state.
- [x] SQLite database lives outside observed repositories in the host state directory.
- [x] Schema version is explicit.
- [x] Repository and execution-session history are projected into SQLite.
- [x] The index is rebuildable from the host catalog.
- [x] Heartbeat-only `last_seen` updates do not cause repeated SQLite snapshot writes.
- [x] Search matches repository name, path, convention, and agent.
- [x] Search returns repository identity while live catalog state remains UI authority.
- [x] Host page exposes a restrained search UI without changing repository/project hierarchy.
- [x] SQLite failure degrades search independently from host observation.
- [x] SQLite implementation is CGO-free and not OS-specific.
- [ ] `gofmt`, module verification, `go vet`, race tests, build, and release cross-build pass.
- [ ] Real macOS host confirms `index.sqlite` creation and live repository search.

## Out of scope

- replacing `catalog.json` as the compatibility history snapshot;
- FTS5 or semantic/vector search;
- indexing specification titles/content;
- indexing Evidence records;
- indexing GitHub PR text;
- multi-host federation;
- retention/compaction policy for long-term history;
- Acceptance policy.
