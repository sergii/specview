---
specview:
  status: new
---

# Git status observation

Expose useful Git state alongside specification state without turning Specview into a Git client.

## Initial questions

- branch name.
- clean or dirty working tree.
- modified/untracked file counts.
- whether Git state belongs in the header, project summary, or a later activity view.

## Constraint

This is not part of the v0.0.1 vertical slice and must not delay the filesystem observer POC.
