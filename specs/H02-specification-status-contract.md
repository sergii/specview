---
specview:
  status: done
---

# Specification status contract

Parse Markdown files and expose the minimal `new`, `in_progress`, and `done` lifecycle.

## Acceptance

- missing metadata defaults to `new`.
- unknown statuses remain visible as metadata errors.
- Specview never writes status changes.
