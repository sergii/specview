# SpecviewAdapter

`SpecviewAdapter` is the native artifact adapter for Specview repositories.

## Native convention

Specview separates tool metadata from project knowledge.

```text
repo/
├── .specview.yaml       # versioned Specview configuration / adapter selection
├── .specview/           # reserved runtime namespace, cache/index later
└── specs/               # user-owned durable specification artifacts
    ├── H01.md
    ├── H02.md
    └── H03.md
```

The default artifact root is:

```text
specs/
```

`specs/` is ordinary repository content. It is not, by itself, a reliable signature that a repository uses the native Specview format because other SDD systems, including GitHub Spec Kit, also use a top-level `specs/` directory.

Each native Markdown file is projected as a normalized artifact with at least:

```text
id
kind = spec
path
title
declared status
modified time
body
metadata error, if any
```

The current v0 contract reads status from namespaced front matter:

```yaml
---
specview:
  status: in_progress
---
```

A file without status metadata defaults to `new`.

## Configuration and identity

Explicit configuration:

```yaml
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
```

For backward compatibility, an omitted `specs.adapter` in `.specview.yaml` currently resolves to `specview`.

`.specview.yaml` means the repository has explicitly configured Specview. It does not necessarily mean the native adapter forever: the configuration may later select `github-spec-kit`, `openspec`, `kiro`, or another adapter.

The `.specview/` directory is reserved for Specview-owned runtime material such as a rebuildable SQLite projection, indexes, sockets, or transient state. It must not become the canonical store for project intent.

`SpecviewAdapter` is not a synonym for "any directory containing Markdown". A future custom Markdown adapter may support arbitrary company-specific layouts and metadata.

## Detection precedence

Adapter selection follows authority rather than folder-name guessing:

```text
1. .specview.yaml with explicit adapter   -> configured adapter
2. .specview.yaml without adapter         -> SpecviewAdapter (compatibility default)
3. .specify/                              -> GitHub Spec Kit adapter
4. openspec/                              -> OpenSpec adapter
5. .kiro/specs/                           -> Kiro adapter
6. _bmad-output/                          -> BMAD adapter
7. specs/ only                            -> ambiguous, not a Specview signature
```

A plain top-level `specs/` directory may still be observed after explicit configuration, but Specview should not silently claim ownership of its semantics during framework auto-detection.

## Source of truth

`SpecviewAdapter` reads repository files. It does not own canonical project intent and does not write status changes.

A future SQLite projection under `.specview/` may cache and index normalized artifacts, but deleting that runtime state must not remove project knowledge.

## UI independence

The adapter produces normalized input. It has no dependency on list, board, graph, timeline, or any other UI representation.
