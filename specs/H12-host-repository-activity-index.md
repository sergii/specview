---
specview:
  status: done
---

# Host repository activity index

Move Specview one level above the current single-project dashboard.

## Intent

Running `specview` starts a host-level observer. The first page answers: what repositories have I worked in on this host recently, and what is active now?

The minimum top-level project entity is a concrete Git repository. Parent company folders are not top-level entities in this slice.

## Vertical slice

```text
start Specview anywhere
  -> show host dashboard
  -> scan running Codex processes
  -> resolve Codex cwd to canonical Git repository
  -> persist observed repository/session
  -> detect specification convention read-only
  -> render Today / Yesterday / Earlier
  -> click repository
  -> render supported Intent projection
     or explain that no specification pattern is recognized
```

The root page is a one-column Flow-style list. Each row shows repository name/path, active or idle state, agent label, detected specification convention, and relative last activity.

Codex running from a linked Git worktree resolves to the canonical/main worktree so linked worktrees do not become duplicate top-level projects.

Linux discovery uses `/proc`; macOS uses `ps` plus `lsof` for cwd. No Specview API call from Codex is required.

Recognized strong signatures:

```text
.specview.yaml  -> Specview
.specify/       -> GitHub Spec Kit
openspec/       -> OpenSpec
.kiro/specs/    -> Kiro
_bmad-output/   -> BMAD
```

Plain `specs/` remains ambiguous. Kiro and BMAD are detection-only in this slice; existing Specview, GitHub Spec Kit, and OpenSpec adapters continue to provide artifact projections.

Observed host history is stored outside repositories. The POC uses a small JSON catalog under the user's host state directory and is explicitly replaceable by SQLite later.

## Acceptance

- `specview` starts without `.specview.yaml` in the current directory;
- host name is rendered;
- no full-disk repository crawl occurs;
- running Codex in a Git repository makes that repository appear;
- linked worktree activity resolves to one top-level repository;
- repository/session history survives restart;
- repositories group into Today, Yesterday, and Earlier;
- empty Today state is visible;
- all five strong specification signatures are detected read-only;
- plain `specs/` does not force a convention;
- unrecognized repositories remain visible with an explanatory message;
- supported repository adapters still render existing project work;
- Kiro/BMAD detection does not pretend parser support exists;
- host activity refreshes through SSE;
- project files are never created by host discovery.

## Verification

The implementation passed the GitHub Actions gates:

```text
gofmt                         PASS
go mod tidy / module hygiene  PASS
go vet ./...                   PASS
go test -race ./...            PASS
go build ./cmd/specview        PASS
release cross-build            PASS
```

Release cross-build covers Linux and macOS on amd64 and arm64.

## Non-goals

SQLite migration, worktree UI, company hierarchy, recursive repository discovery, repository search, Claude/OpenCode auto-discovery, multi-host aggregation, and Acceptance policy.

## References

- `docs/decisions/ADR-003-host-repository-session-model.md`
