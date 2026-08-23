# Specview contract fixtures

Files in this directory are language-neutral conformance fixtures for versioned Specview formats.

They are product contracts, not Go-specific test helpers.

Current fixtures:

```text
config/v1-acceptance.yaml   .specview.yaml version 1
 evidence/v1-passed.json    native Evidence version 1
 catalog/v1.json             host catalog persistence version 1
```

The current Go implementation must consume these fixtures in CI. A future Rust implementation must consume the same fixtures before it can replace the Go implementation.

Rules:

- Do not edit an existing fixture to change the meaning of a released version.
- Add a new fixture for a new compatible case.
- Introduce a new contract version or explicit migration when semantics are incompatible.
- Future public CLI JSON, MCP schemas, and federation formats must add their own versioned fixtures when they become contracts.

These fixtures intentionally do not freeze HTML, CSS, Go structs, package names, SQLite schema internals, or a future graph wire format that has not yet been designed.
