---
specview:
  status: in_progress
---

# Ephemeral demo recipe

Replace persistent demo state with an agent-executable, disposable demonstration scenario.

## Acceptance

- canonical recipe lives in `demo.md`.
- agent creates a unique OS temporary directory using the prefix `specview-demo-`.
- suffix is an opaque session identifier and must not encode username, hostname, device name, repository name, Git SHA, or other personal information.
- demo creates its own `.specview.yaml` and `specs/`.
- demo starts with 2 done, 2 in_progress, and 2 new specifications.
- agent pauses after setup so the user can start Specview.
- `run demo` performs visible status transitions with pauses.
- demo directory persists after the animation.
- only explicit `cleanup` removes the temporary directory.
- multiple demo sessions can coexist without shared state.
- Specview binary contains no demo-specific code path or bundled demo data.