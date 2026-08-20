---
specview:
  status: in_progress
---

# Configurable project root

Allow `.specview.yaml` to point Specview at a project directory other than the directory containing the configuration file.

## Contract

```yaml
project:
  name: ""
  root: "."
```

## Acceptance

- `project.root` defaults to `.` when omitted.
- relative roots resolve from the directory containing `.specview.yaml`.
- absolute roots are supported.
- `specs.path` resolves relative to `project.root`.
- the observed project root must already exist and be a directory.
- the feature is generic and must not introduce demo-specific behavior.
