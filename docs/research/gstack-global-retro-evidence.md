# gstack global retro as product evidence

- Status: research note
- Date: 2026-09-05
- Source: gstack `/retro global`
- Upstream: https://github.com/garrytan/gstack/blob/main/retro/SKILL.md.tmpl

## Observation

gstack has a `global` retrospective mode that works across projects rather than inside one repository. It discovers AI coding sessions, associates them with Git repositories, aggregates Git activity, and produces a cross-project engineering retrospective.

Its current global report includes signals such as:

- active projects;
- commits and LOC across repositories;
- AI coding sessions split by tool;
- active days and shipping streaks;
- per-project contribution breakdowns;
- context switches per day;
- cross-project insights;
- trends against previous retrospectives.

A public example shared by John Nunemaker showed the important product behavior clearly: the useful output was not only raw Git statistics. The retrospective inferred a higher-level work pattern - every active day touched multiple repositories and the dominant pattern was context switching. In the shared week, the daily breadth ranged from four to nine repositories.

## Why this is strong evidence for Specview

This validates a problem that already exists inside Specview's direction: useful software-development state is not bounded by one repository.

The important user question is often not:

> What changed in this repository?

It is:

> What have I been working on across my whole development workspace, how fragmented was the work, and where did the effort go?

GitHub, Git history, and repo-local dashboards can expose the underlying events, but a cross-repository projection can turn those events into a model of engineering work.

This is evidence for the value of Specview's normalized graph and multi-repository model, not evidence that Specview should become a clone of gstack. gstack's retro is primarily retrospective analytics. Specview's broader opportunity is to connect retrospective activity with current work state, intent, execution sessions, evidence, acceptance, hosts, worktrees, and future actions.

## Product insight

Add a first-class **Retro / Focus** projection to the candidate Specview views.

Conceptually:

```text
normalized workspace graph
        |
        +--> List
        +--> Kanban
        +--> Graph
        +--> Timeline
        +--> Activity stream
        +--> Retro / Focus
```

The Retro projection should answer both historical and transition questions:

```text
Where did my time/activity go?
How many projects did I touch?
How fragmented was the work?
Which initiatives dominated the period?
Which work remained unfinished?
Which active branches/worktrees/PRs did the activity leave behind?
Which agent sessions produced uncommitted or unmerged changes?
What changed since the previous period?
```

This is a natural extension of ADR-006, which already defines Specview views as replaceable projections over normalized domain state.

## Candidate metrics

Initial metrics can be derived from facts Specview already has or is expected to normalize:

```text
active repositories
active work items
active worktrees
commits
PRs opened / merged / waiting
execution sessions by agent/tool
session-to-work-item correlation
context switches per day
repos touched per day
work items touched per day
longest single-project focus interval
unfinished branches/worktrees
stale active work
uncommitted agent output
acceptance/evidence transitions
```

Later versions can add higher-level derived metrics such as:

```text
focus share by project / initiative
fragmentation score
handoff count between repositories
unfinished-work carryover
blocked-work duration
activity-to-ship ratio
agent-session-to-accepted-change ratio
```

Derived metrics must remain projections over observed facts. Specview should avoid pretending that commit count or LOC equals productivity.

## A useful distinction from gstack

The strongest differentiation is temporal continuity.

gstack global retro is approximately:

```text
past activity
  -> aggregate
  -> retrospective
  -> insight
```

Specview can become:

```text
past activity + current workspace state + intent + execution + evidence
  -> normalized graph
  -> retrospective + current-state projections
  -> insight + next-action context
```

That allows a retrospective to end with current operational state instead of only a summary of the past. For example:

```text
9 repositories touched this week
4 active initiatives
17 agent sessions
6 unfinished branches
3 PRs waiting for review
2 blocked work items
11 context switches
longest uninterrupted project focus: 74 min
3 agent sessions produced changes not yet committed
```

The exact numbers and heuristics are future design work, but this is the product direction worth preserving.

## CLI/UI hypothesis

Possible CLI surfaces:

```text
specview retro
specview retro week
specview retro --since 7d
specview retro global
```

Possible UI names:

- Retro
- Focus
- Worklog
- Activity intelligence

`Retro` is the clearest initial name because it describes the time-oriented projection without changing the underlying domain model.

## Implication

Treat gstack `/retro global` as external evidence that cross-project engineering observability is useful and understandable to developers.

Do not copy its report wholesale. Use it as validation for a Specview projection that connects historical activity to the live development graph.

## Follow-up

Before implementation:

1. define the minimum normalized temporal facts required for a Retro projection;
2. identify which facts already exist in `ExecutionSession`, `RepositoryInstance`, `Worktree`, `WorkItem`, Git, and Forge projections;
3. prototype a read-only weekly Retro from existing data;
4. test whether context-switching and unfinished-work insights remain useful without inventing productivity scores;
5. decide whether Retro belongs in the first post-v0.0.1 view expansion or a later analytics slice.
