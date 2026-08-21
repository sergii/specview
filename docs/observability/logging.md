# Logging and development diagnostics

Specview uses the standard library `log/slog` API for application diagnostics.

Runtime logs are intentionally **disabled by default**. Starting the dashboard should not fill a user's terminal with scanner, HTTP, SSE, SQLite, or process-discovery messages.

## CLI logging modes

Use an explicit flag when diagnostics are needed:

```bash
specview --verbose
specview --debug
specview --log-level=warn
```

The levels are:

```text
default     logs disabled
--verbose   info
--debug     debug + source locations
--log-level debug|info|warn|error
```

Logging flags are global and may be placed before or after the command:

```bash
specview --debug
specview serve --debug
specview doctor --verbose
```

`-v` remains the existing short form for `--version`; Specview deliberately uses the unambiguous long form `--verbose` for verbose logging.

Development console output is rendered through `github.com/lmittmann/tint`. JSON mode uses `slog.JSONHandler`, so fields remain machine-readable for future OpenTelemetry or log-pipeline integration.

Example debug output:

```text
20:52:31.104 DBG Codex process matched app=specview version=dev pid=15732
20:52:31.118 DBG Codex cwd resolved app=specview version=dev pid=15732 cwd=/Users/.../wms
20:52:31.126 DBG Codex discovery succeeded app=specview version=dev pid=15732 repository=/Users/.../wms
```

## Environment variables

Environment variables remain useful for services, CI, and persistent shell configuration:

```text
SPECVIEW_LOG_LEVEL=debug|info|warn|error
SPECVIEW_LOG_FORMAT=console|json
SPECVIEW_LOG_SOURCE=true|false
SPECVIEW_LOG_COLOR=true|false
SPECVIEW_ENV=production
NO_COLOR=1
```

`SPECVIEW_LOG_LEVEL` enables runtime logging. Without a level, logging stays disabled.

Precedence is:

```text
CLI logging flag
  > SPECVIEW_LOG_LEVEL
  > disabled default
```

`SPECVIEW_ENV=production` changes the default format to JSON when logging is enabled; it does not enable logging by itself. For example:

```bash
SPECVIEW_ENV=production SPECVIEW_LOG_LEVEL=info specview
```

Explicit `SPECVIEW_LOG_FORMAT`, `SPECVIEW_LOG_SOURCE`, and `SPECVIEW_LOG_COLOR` values still override their format-specific defaults.

## Logged areas

Debug logging intentionally covers the current vertical slice broadly:

- process start and selected command;
- host state path and server startup/shutdown;
- host activity refreshes;
- Darwin/Linux Codex process matching;
- cwd discovery;
- canonical Git repository resolution and fallback;
- produced observations and skipped discovery stages;
- SSE subscriptions and broadcasts;
- HTTP request start/completion;
- SQLite host-index activity;
- doctor diagnostics.

Do not add secrets, access tokens, authorization headers, environment dumps, or file contents to routine logs. Full process command lines remain visible in the explicit `doctor` human-readable report because that command is an intentional local diagnostic action; routine scanner logs use PID/path/stage fields instead.

## Developer commands

Canonical development update/run command:

```bash
bin/dev install
```

Shortcut:

```bash
bin/install
```

Both now start Specview with the same quiet runtime default as the released binary. Pass logging flags through when needed:

```bash
bin/dev install --verbose
bin/install --debug
```

The development command performs, in order:

```text
git fetch --prune origin
  -> git pull --ff-only
  -> gofmt check
  -> go mod tidy + verify
  -> go vet ./...
  -> go test -race ./...
  -> build ./bin/specview
  -> run Specview with the requested logging mode
```

Other commands:

```bash
bin/dev check
bin/dev build
bin/doctor
```

`bin/doctor` always rebuilds the current checkout before running `specview doctor`, preventing diagnostics from accidentally using a stale local binary. The doctor report itself is human-facing command output and remains visible even when runtime logs are disabled.
