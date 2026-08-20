---
specview:
  status: in_progress
---

# Companion demo repository

Keep the Demo Project in `https://github.com/sergii/specview-demo` instead of embedding demo data in the Specview binary.

## Acceptance

- independent Git history.
- `.specview.yaml` with `project.name: "Demo Project"` and `project.demo: true`.
- exactly 10 specifications with multiple statuses.
- a small real implementation and tests.
- clonable and usable with coding agents like any ordinary repository.
