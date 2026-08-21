# Logging and development diagnostics

Specview uses the standard library `log/slog` API for all application logs.

Development console output is rendered through `github.com/lmittmann/tint`, a colorized `slog.Handler`. Production mode keeps the same logging calls and swaps the handler to `slog.JSONHandler`, so fields remain machine-readable for future OpenTelemetry or log-pipeline integration.

## Development defaults

```text
format: console
level:  debug
source: on
color:  on
```

Example shape:

```text
20:52:31.104 DBG Codex process matched app=specview version=dev pid=15732
20:52:31.118 DBG Codex cwd resolved app=specview version=dev pid=15732 cwd=/Users/.../wms
20:52:31.126 DBG Codex discovery succeeded app=specview version=dev pid=15732 repository=/Users/.../wms
```

The exact colors are terminal-dependent.

## Production defaults

Set:

```bash
SPECVIEW_ENV=production specview
```

Production defaults to JSON at `info` level:

```json
{"time":"...","level":"INFO","msg":"Specview host observer started","app":"specview","version":"...","hostname":"devbox-01","state":"...","address":"http://127.0.0.1:7331"}
```

The JSON format is intended to be ingestible by a future OTEL collector, journald bridge, Loki, Vector, Fluent Bit, or another observability pipeline without changing domain logging calls.

## Environment variables

```text
SPECVIEW_LOG_FORMAT=console|json
SPECVIEW_LOG_LEVEL=debug|info|warn|error
SPECVIEW_LOG_SOURCE=true|false
SPECVIEW_LOG_COLOR=true|false
SPECVIEW_ENV=production
NO_COLOR=1
```

`SPECVIEW_ENV=production` only supplies defaults. Explicit `SPECVIEW_LOG_*` values win.

## Logged areas

Development debug logging intentionally covers the current vertical slice broadly:

- process start and selected command;
- host state path and server startup/shutdown;
- every host activity refresh;
- Darwin/Linux Codex process matching;
- cwd discovery;
- canonical Git repository resolution and fallback;
- produced observations and skipped discovery stages;
- SSE subscriptions and broadcasts;
- HTTP request start/completion;
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

The command performs, in order:

```text
git fetch --prune origin
  -> git pull --ff-only
  -> gofmt check
  -> go mod tidy + verify
  -> go vet ./...
  -> go test -race ./...
  -> build ./bin/specview
  -> run Specview with console/debug logging
```

Other commands:

```bash
bin/dev check
bin/dev build
bin/doctor
```

`bin/doctor` always rebuilds the current checkout before running `specview doctor`, preventing diagnostics from accidentally using a stale local binary.
