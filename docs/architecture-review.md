# Architecture Review

External code review findings and architectural insights for GoKart.

---

## Architecture Strengths

### Thin Wrappers Philosophy

The core design principle of providing thin wrappers around battle-tested packages is working well in practice. Factory functions return underlying types directly (`*pgxpool.Pool`, `chi.Router`, `*redis.Client`) rather than custom abstractions, giving users direct access to the full API surface of wrapped dependencies.

### SQLite Implementation

The SQLite integration demonstrates excellent defaults:

- **Zero-CGO**: Uses `modernc.org/sqlite` for pure-Go implementation
- **WAL Mode**: Write-Ahead Logging enabled by default for better concurrency
- **Pragmas**: Proper pragma defaults configured for production use
- **Safety**: Transaction helpers with automatic rollback on panic

### CLI Wrapper Impact

The `cli` subpackage wrapper around `cobra` + `lipgloss` successfully reduces boilerplate:

- **~50 lines saved** per project through sensible defaults
- Styled output, tables, spinners provided out of box
- Editor integration (`CaptureInput`) identified as premium CLI feature

### Transaction Safety

The closure-based transaction pattern handles edge cases correctly:

```go
gokart.WithTransaction(ctx, pool, func(tx pgx.Tx) error { ... })
gokart.SQLiteTransaction(ctx, db, func(tx *sql.Tx) error { ... })
```

**Key safety feature**: Automatic rollback on panic, not just error returns.

### Graceful Shutdown

The HTTP server implementation includes production-ready graceful shutdown patterns, properly handling in-flight requests during termination.

---

## Design Decisions Documented

### Root Package Alias Removal (v1.0)

**Decision**: Root package aliases will be removed before v1.0 release.

**Rationale**: Forces users to import from specific subpackages (e.g., `gokart/postgres`, `gokart/cache`), enabling better dead code elimination by the linker.

**Migration path**: Update imports to reference subpackages directly.

### Global Mutex for Migrations

**Pattern**: Global mutex protects migration operations.

**Justification**: Goose (`pressly/goose/v3`) uses package-level configuration variables. The mutex prevents race conditions when multiple goroutines attempt concurrent migrations or configuration changes.

**Trade-off**: Serializes migration operations, but migrations are infrequent and safety-critical.

### Validator Tag Defaults

**Configuration**: Validator uses JSON tag names by default.

**Good for**: API development, JSON-heavy applications.

**Caveat**: CLI applications may need explicit struct tags if field names differ from desired CLI flags.

**Override**: Configure validator to use different tag names when appropriate for domain.

### Editor Temporary File Permissions

**Security improvement**: `CaptureInput` now creates temporary files with explicit `0600` permissions (owner read/write only).

**Context**: Prevents other users on multi-user systems from reading potentially sensitive data being edited.

---

## Dependency Considerations

### go.mod Size vs. Binary Size

**Observation**: `go.mod` contains many dependencies, which may appear heavy.

**Mitigation strategy**:

1. **Subpackage organization**: Dependencies isolated to specific subpackages
2. **Dead code elimination**: Go linker strips unused code at build time
3. **Import selectivity**: Users only import what they need (e.g., `gokart/postgres` without `gokart/cache`)

**Result**: Final binary size reflects actual usage, not total dependency graph.

**Example**:
```go
// Only pulls in pgx dependencies
import "gokart/postgres"

// Doesn't include redis, openai, sqlite, etc.
```

---

## Best Patterns Identified

### 1. SQLite Package (`gokart/sqlite`)

**Excellence factors**:

- Zero-CGO implementation (pure Go, cross-compilation friendly)
- WAL mode enabled by default
- Production-ready pragma configuration
- Transaction helpers with panic recovery

**Use when**: Building CLI tools, embedded databases, or services requiring SQLite without CGO complexity.

### 2. Transaction Closure Pattern

**Pattern**:
```go
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error
```

**Advantages**:

- Automatic rollback on error
- **Panic safety**: Recovers and rolls back on panic
- Prevents forgot-to-commit/rollback bugs
- Clean API surface (no manual Begin/Commit/Rollback)

**Use when**: Any database transaction work. Prefer this over manual transaction management.

### 3. CaptureInput Editor Bridge (`gokart/cli`)

**Feature**:
```go
text, err := cli.CaptureInput("# Enter description", "md")
```

**Premium aspects**:

- Opens `$EDITOR` for long-form input
- Returns edited content to application
- Secure temp file handling (0600 permissions)
- Markdown/text syntax support

**Use when**: CLI needs multi-line input, descriptions, commit messages, or any scenario where readline is insufficient.

**Inspiration**: Git commit message editing pattern.

---

## Recommendations for Developers

### When Reviewing Codebase

1. **Understand the wrapper philosophy**: Code should be thin. Business logic belongs in user applications.
2. **Check return types**: Factory functions should return underlying types, not custom wrappers.
3. **Verify defaults**: New components should have sensible, production-ready defaults.

### When Adding Components

1. **Fight for inclusion**: If stdlib can do it, keep it in stdlib. No helpers for things Go already does well.
2. **Isolate dependencies**: New dependencies should go in subpackages when possible.
3. **Follow transaction pattern**: Closure-based patterns with automatic cleanup preferred.

### When Using GoKart

1. **Import selectively**: Only import subpackages you need (linker will strip the rest).
2. **Trust the defaults**: Pragmas, timeouts, retry logic already configured for production.
3. **Use transaction helpers**: Don't manually manage Begin/Commit/Rollback when helpers exist.

---

## Version Roadmap Impact

### Pre-v1.0

- Remove root package aliases (breaking change)
- Force subpackage imports for better tree shaking

### Post-v1.0

- API stability guaranteed
- Dependency updates non-breaking (except security fixes)
