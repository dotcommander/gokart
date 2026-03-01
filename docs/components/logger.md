# Logger

Structured logging for Go applications built on [log/slog](https://pkg.go.dev/log/slog). Provides zero-config defaults, text and JSON formats, and a file logger designed for TUI applications where stdout must stay clean.

## Installation

```bash
go get github.com/dotcommander/gokart/logger
```

## Quick Start

```go
import "github.com/dotcommander/gokart/logger"

// Zero-config: info level, JSON format, stderr
log := logger.NewDefault()
log.Info("server started", "port", 8080)
log.Error("database unavailable", "err", err, "host", "localhost")
```

---

## Configuration

### Config Struct

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Level` | `string` | `"info"` | Log level: `debug`, `info`, `warn`, `error` |
| `Format` | `string` | `"json"` | Output format: `json`, `text` |
| `Output` | `io.Writer` | `os.Stderr` | Destination writer |

### Log Levels

| Level | Constant | When to use |
|-------|----------|-------------|
| `"debug"` | `slog.LevelDebug` | Verbose diagnostics during development |
| `"info"` | `slog.LevelInfo` | Normal operational events |
| `"warn"` | `slog.LevelWarn` | Unexpected state that is recoverable |
| `"error"` | `slog.LevelError` | Failures that require attention |

Any unrecognized level string falls back to `info`.

---

## Functions

### `New(cfg Config) *slog.Logger`

Creates a logger with explicit configuration. Returns a `*slog.Logger` directly — no wrapper type.

```go
// Development: human-readable text to stderr
log := logger.New(logger.Config{
    Level:  "debug",
    Format: "text",
})
log.Debug("query executed", "sql", "SELECT * FROM users", "rows", 42)
```

```go
// Production: JSON to stderr (default format)
log := logger.New(logger.Config{
    Level: "warn",
})
log.Warn("rate limit approaching", "current", 95, "limit", 100)
```

```go
// Custom output: write to an existing writer
var buf bytes.Buffer
log := logger.New(logger.Config{
    Level:  "debug",
    Format: "json",
    Output: &buf,
})
```

---

### `NewDefault() *slog.Logger`

Creates a logger with zero-config defaults: `info` level, JSON format, writing to `os.Stderr`. Use this when you want structured logging without any setup.

```go
log := logger.NewDefault()
log.Info("application started", "version", "1.2.3")
log.Info("request handled", "method", "GET", "path", "/api/users", "status", 200, "latency_ms", 14)
```

---

### `NewFile(appName string) (*slog.Logger, func(), error)`

Creates a logger that appends JSON-formatted logs to `/tmp/<appName>.log`. Always runs at `debug` level — the file captures everything.

This is the correct logger for TUI applications, CLI tools with progress output, and any program where writing to stderr would corrupt the terminal display.

```go
log, cleanup, err := logger.NewFile("myapp")
if err != nil {
    return fmt.Errorf("open log file: %w", err)
}
defer cleanup()

log.Info("application started")
log.Debug("config loaded", "path", cfgPath)

// Tail the log from another terminal while the app runs:
// tail -f /tmp/myapp.log
```

The returned cleanup function closes the underlying file. Always defer it immediately after checking the error.

**Log file location:** `os.TempDir()/<appName>.log`

| Platform | Path |
|----------|------|
| macOS | `/tmp/myapp.log` |
| Linux | `/tmp/myapp.log` |
| Windows | `%TEMP%\myapp.log` |

---

### `Path(appName string) string`

Returns the path where `NewFile` writes logs, without opening the file. Useful for printing the log location at startup or in help text.

```go
log, cleanup, err := logger.NewFile("myapp")
if err != nil {
    return err
}
defer cleanup()

fmt.Fprintf(os.Stderr, "Logging to %s\n", logger.Path("myapp"))
// Output: Logging to /tmp/myapp.log
```

---

## Best Practices

### Use `NewFile` for TUI and interactive CLI tools

Any program that renders to the terminal — progress bars, spinners, interactive prompts — must not write log lines to stderr or stdout, as they corrupt the display. Route all logging to a file and tail it during development:

```go
func main() {
    log, cleanup, err := logger.NewFile("mytui")
    if err != nil {
        fmt.Fprintln(os.Stderr, "warn: logging disabled:", err)
        log = logger.NewDefault() // fallback
    } else {
        defer cleanup()
    }

    // Pass log down through dependency injection
    app := newApp(log)
    app.Run()
}
```

```bash
# In a second terminal while the app runs
tail -f /tmp/mytui.log
```

### Pass the logger as a dependency

Do not store the logger in a package-level variable. Pass it explicitly through constructors or a dependencies struct. This makes components testable and keeps coupling explicit.

```go
// Right
type Server struct {
    log *slog.Logger
    db  *sql.DB
}

func NewServer(log *slog.Logger, db *sql.DB) *Server {
    return &Server{log: log, db: db}
}
```

### Add context to log calls with structured attributes

Prefer key-value pairs over formatted strings. They remain queryable in log aggregators.

```go
// Right
log.Error("query failed", "table", "users", "err", err, "duration_ms", elapsed.Milliseconds())

// Wrong — loses structure
log.Error(fmt.Sprintf("query on users failed after %dms: %v", elapsed.Milliseconds(), err))
```

### Set level from configuration

Expose the log level as a config field so operators can raise verbosity without a rebuild.

```go
type AppConfig struct {
    LogLevel string `mapstructure:"log_level"`
}

cfg, _ := gokart.LoadConfig[AppConfig]("config.yaml")
log := logger.New(logger.Config{Level: cfg.LogLevel})
```

---

## Reference

| Export | Kind | Description |
|--------|------|-------------|
| `Config` | struct | Logger configuration: level, format, output |
| `New(cfg Config)` | func | Creates logger with explicit config |
| `NewDefault()` | func | Creates logger with info/JSON/stderr defaults |
| `NewFile(appName string)` | func | Creates file logger at `/tmp/<appName>.log` |
| `Path(appName string)` | func | Returns log file path without opening it |

---

## See Also

- [Config](/api/gokart) - Loading application configuration
- [State](/api/gokart) - Persisting application state between runs
- [CLI](/api/cli) - Building styled CLI applications
- [log/slog package](https://pkg.go.dev/log/slog) - Standard library structured logging
