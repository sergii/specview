# SpecviewAdapter

`SpecviewAdapter` is the native artifact adapter for Specview repositories.

## Native convention

The default root is:

```text
specs/
```

The minimal layout is intentionally lightweight:

```text
specs/
  H01.md
  H02.md
  H03.md
```

Each Markdown file is projected as a normalized artifact with at least:

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

## Configuration

Explicit configuration:

```yaml
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
```

For backward compatibility, an omitted `specs.adapter` currently resolves to `specview`.

## Adapter identity

`SpecviewAdapter` is not a synonym for "any directory containing Markdown".

A future custom Markdown adapter may support arbitrary company-specific layouts and metadata. Keeping that concern separate prevents the native Specview contract from becoming ambiguous.

## Detection precedence

Other SDD frameworks may also use `specs/`. GitHub Spec Kit is the important example.

Future zero-config detection must therefore prefer specific framework signatures before falling back to `SpecviewAdapter`:

```text
.specify/          -> GitHub Spec Kit adapter
openspec/          -> OpenSpec adapter
.kiro/specs/       -> Kiro adapter
_bmad-output/      -> BMAD adapter
specs/             -> SpecviewAdapter fallback
```

Explicit configuration always wins over detection.

## Source of truth

`SpecviewAdapter` reads repository files. It does not own canonical project intent and does not write status changes.

A future SQLite projection may cache and index the normalized artifacts, but deleting that database must not remove project knowledge.

## UI independence

The adapter produces normalized input. It has no dependency on list, board, graph, timeline, or any other UI representation.
