# Test a GoKart application

```go
var out bytes.Buffer
var cli CLI
parser, err := kong.New(&cli, kong.Writers(&out, &out), kong.BindTo(t.Context(), (*context.Context)(nil)))
if err != nil { t.Fatal(err) }
parsed, err := parser.Parse([]string{"counter", "--by", "2"})
if err != nil { t.Fatal(err) }
if err := parsed.Run(testContext); err != nil { t.Fatal(err) }
```

Write command results through `*kong.Context.Stdout` and diagnostics through `*kong.Context.Stderr`. Construct the parser with `kong.Writers` so tests replace both streams without mutating process globals.

## Command and action tests

Construct the smallest typed command tree, parse an explicit argument slice, pass dependencies to `parsed.Run`, and assert the buffer and returned error. Do not call `main`: it owns `os.Exit` and sits outside unit tests. For root configuration, construct the root and use a temporary config file plus injected or test-scoped environment state.

Test actions as ordinary Go. Pass typed input and the narrowest dependency, then cover validation, success, and failure without Kong. The generated greet test demonstrates this boundary.

## Temporary SQLite

```go
path := filepath.Join(t.TempDir(), "test.db")
db, err := sqlite.OpenContext(t.Context(), path)
if err != nil { t.Fatal(err) }
t.Cleanup(func() { _ = db.Close() })
```

Apply production migrations, then test actions against `db`. A temporary file exercises file-backed behavior. Use `:memory:` only when its one-connection, memory-only semantics are intended. Test rollback by returning a sentinel error and asserting that no partial rows remain.

## HTTP handlers

```go
req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(`{"body":"hi"}`))
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)
```

Cover malformed JSON, body limits, validation errors, dependency failures, status, and content type without opening a port.

## PostgreSQL and Redis boundaries

Keep command and business tests independent of live services through consumer-owned repository interfaces. Put pgx, migration, Redis expiration, and connection behavior in an integration-test lane with explicit service setup and cleanup. Missing services are environment skips, not unit-test success.

## Generator verification

Every new scaffold prepares pinned dependencies with `go get` and `go mod tidy`.
Unless disabled, GoKart then runs `go test ./...` and `go build ./...` for flat,
structured, and integration projects alike. `--no-verify` skips only tests and
build. `--verify-only` tidies, tests, and builds an existing project. The
default timeout is `5m`; `--verify-timeout 0` disables it.

Before shipping:

```bash
go test ./...
go build -o notes .       # flat project
go build -o notes ./cmd   # structured project
```
