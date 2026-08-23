---
specview:
  status: done
---

# Filesystem observation

Detect changes under the configured specs directory and refresh the browser automatically.

## Acceptance

- recursive Markdown discovery.
- lightweight polling snapshot is acceptable for the POC.
- browser updates through Server-Sent Events without manual refresh.
- SIGINT and SIGTERM stop the server cleanly even while an SSE client is connected.
