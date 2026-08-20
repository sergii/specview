---
specview:
  status: done
---

# Demo isolation

Keep demo data and demo execution outside the Specview binary and outside the user's existing project.

## Acceptance

- Specview binary embeds no demo specifications.
- Specview has no `--demo`, `demo` command, or demo-specific configuration flag.
- public demo is an agent-executable recipe rather than persistent mutable state.
- demo sessions use isolated OS temporary directories.
- persistent `sergii/specview-demo` may exist only as a temporary development/integration fixture and is not required by the public demo flow.
