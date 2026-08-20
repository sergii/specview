# Specview ephemeral demo

This file is an agent-executable demonstration recipe for Specview.

The demo is disposable. It must not modify the user's current repository and it must not depend on a persistent demo repository.

## Phase 1 - prepare

1. Create a unique temporary directory using the operating system's standard temporary-directory mechanism.
2. Use the prefix `specview-demo-` and let the OS generate the suffix.
3. Treat the suffix as an opaque session identifier. Do not derive it from a username, hostname, device name, repository name, Git SHA, email address, or other personal information.
4. Inside that directory create `.specview.yaml` with:

```yaml
version: 1

project:
  name: "Specview Demo"
  root: "."

specs:
  path: specs
  pattern: "*.md"

server:
  host: 127.0.0.1
  port: 7331
```

5. Create `specs/` and these six Markdown specifications:

- `01-project-setup.md` - `done`
- `02-database-schema.md` - `done`
- `03-authentication.md` - `in_progress`
- `04-api-endpoints.md` - `in_progress`
- `05-background-jobs.md` - `new`
- `06-deployment.md` - `new`

Each file must use this front matter:

```markdown
---
specview:
  status: new
---
```

Use the appropriate status for each file and a matching H1 title.

6. Do not initialize Git, do not commit, and do not modify files outside the temporary directory.
7. Print the absolute temporary directory path and the command the user can run to observe it:

```bash
cd <temporary-directory> && specview
```

8. Stop and wait for the user to say `run demo`.

## Phase 2 - run

After the user says `run demo`, perform exactly these transitions, one at a time:

1. `05-background-jobs.md`: `new` -> `in_progress`
2. wait 3 seconds
3. `03-authentication.md`: `in_progress` -> `done`
4. wait 3 seconds
5. `06-deployment.md`: `new` -> `in_progress`
6. wait 3 seconds
7. `06-deployment.md`: `in_progress` -> `done`

Only modify `specview.status` in YAML front matter during the transitions.

After every transition print the file name and new status.

When the sequence finishes, print `Specview demo complete` and keep the temporary directory intact.

## Phase 3 - cleanup

Do not remove the temporary directory automatically.

Only after the user explicitly says `cleanup`, remove the exact temporary directory created for this demo session and report that it was removed.

## Invariants

- The demo must be repeatable.
- Multiple demo sessions may run concurrently without sharing state.
- The demo directory name follows `specview-demo-<opaque-session-id>` semantics.
- The opaque session identifier contains no user or host identity.
- The Specview binary has no demo-specific behavior or bundled demo dataset.
- Specview remains a read-only observer; the agent creates and changes the files.