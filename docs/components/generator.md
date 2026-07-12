# Generator — Project Scaffolding

Scaffold new Go CLI projects and add integrations without re-scaffolding.

## Install

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.10.2
```

Use `@latest` instead only if you intentionally want the newest published
version rather than a reproducible install.

---

## `gokart new`

Creates a new Go project in a subdirectory of the current working directory. Accepts a project name or a path (e.g., `./myapp`, `/tmp/myapp`). The project name is derived from the final path segment.

```bash
gokart new <project-name> [flags]
gokart new cli <project-name> [flags]   # explicit preset (same output)
gokart init <project-name>              # alias for new
gokart create <project-name>            # alias for new
```

### Basic Examples

```bash
# Structured project (default)
gokart new mycli

# Explicit preset form — same output as above
gokart new cli mycli

# Custom module path
gokart new mycli --module github.com/myorg/mycli

# Preview what would be written, without writing anything
gokart new mycli --dry-run

# Preview and verify the scaffold compiles
gokart new mycli --dry-run --verify

# Generate and immediately verify
gokart new mycli --verify

# Check if an already-generated project still compiles
gokart new mycli --verify-only

# CI-friendly machine-readable output
gokart new mycli --dry-run --json
```

### Presets: Structured vs. Flat

**Structured** (default) creates a multi-package project with a `cmd/` entry point and `internal/` packages. Plain structured scaffolds are local-only and unmanaged by default: no global config bootstrap and no `.gokart-manifest.json`.

**Flat** (`--flat`) creates a single `main.go` file. Useful for quick scripts. Combining `--flat` with integration flags (`--db sqlite`, `--db postgres`, `--ai`, `--redis`) is an error — flat projects don't support integrations.

```bash
gokart new mycli            # structured
gokart new mycli --flat     # flat single-file
```

### Generated File Tree

#### Structured (default)

```
mycli/
├── .gitignore
├── README.md                          # Build commands
├── cmd/
│   └── main.go                        # Entry point
├── internal/
│   └── commands/
│       └── root.go                    # Cobra root command
└── go.mod
```

With `--global` or an integration flag, `gokart new` writes `.gokart-manifest.json` so `gokart add` can safely patch generated wiring later. `--global` also adds `internal/app/config.go`. Integration flags add `internal/app/context.go`.

With `--example`:

```
internal/
├── commands/
│   └── greet.go                       # Example command
└── actions/
    ├── greet.go                       # Business logic (testable)
    └── greet_test.go                  # Example test
```

#### Flat (`--flat`)

```
mycli/
├── main.go
└── go.mod
```

### Integration Flags

These flags wire a data store or API client into the project at generation time. They add the import, the dependency to `go.mod`, and the wiring in `internal/app/context.go` and `internal/commands/root.go`.

| Flag | Package added | What it wires |
|------|---------------|---------------|
| `--db sqlite` | `github.com/dotcommander/gokart/sqlite` | `*sql.DB` via `sqlite.Open` |
| `--db postgres` | `github.com/dotcommander/gokart/postgres`, `github.com/jackc/pgx/v5` | `*pgxpool.Pool` via `postgres.Open` |
| `--ai` | `github.com/dotcommander/gokart/ai`, `github.com/openai/openai-go/v3` | `openai.Client` via `ai.NewOpenAIClientWithKey` |
| `--redis` | `github.com/dotcommander/gokart/cache`, `github.com/redis/go-redis/v9` | `*cache.Cache` via `cache.Open` |

Flags may be combined:

```bash
gokart new mycli --db postgres --ai
gokart new mycli --db sqlite --example
gokart new mycli --redis
gokart new mycli --redis --db postgres
```

### Config Scope

Controls whether the generated project bootstraps `config.yaml` under the platform user config directory returned by `os.UserConfigDir()`.

```bash
--config-scope auto     # default: local-only
--config-scope local    # no ~/.config/<app>/ bootstrap
--config-scope global   # always bootstrap ~/.config/<app>/config.yaml
```

The shorthand flags `--local` and `--global` are aliases for `--config-scope local` and `--config-scope global`. They cannot be used together, and cannot be combined with `--config-scope`.

The default scope is `local` in both structured and flat mode. Use `--global` when you want generated config-dir bootstrap code.

### Manifest

Plain CLI scaffolds do not write `.gokart-manifest.json`. Gokart writes the manifest only when the scaffold uses managed features: `--global`, `--db sqlite`, `--db postgres`, `--ai`, or `--redis`.

Use `--no-manifest` to suppress the manifest even for managed scaffolds. Without a manifest, `gokart add` cannot safely patch integrations into that project later.

### Conflict Handling

When the target directory already exists and contains files, `gokart new` checks each file it would write against the manifest. The default behavior is to fail and report all conflicting paths.

| Flag | Behavior |
|------|----------|
| _(default)_ | Fail on first conflicting file; report all conflicts |
| `--force` | Overwrite conflicting files |
| `--skip-existing` | Keep existing files; write only missing files |

`--force` and `--skip-existing` cannot be combined.

### Verification Flags

| Flag | Behavior |
|------|----------|
| `--verify` | After scaffolding, run `go mod tidy` then `go test ./...` in the project directory |
| `--verify-only` | Skip scaffolding; run verification only against the existing target directory |
| `--verify-timeout <duration>` | Maximum time allowed for verification commands (default `5m`; `0` disables the timeout) |

`--verify-only` cannot be combined with `--dry-run`. Generation flags (`--flat`, `--db`, `--ai`, `--example`, `--config-scope`, `--force`, `--skip-existing`, `--no-manifest`) are ignored when `--verify-only` is set.

When `--dry-run --verify` is used together, the scaffolder writes to a temporary directory, verifies, then removes it. No files are written to the target.

### All `gokart new` Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--flat` | bool | false | Single `main.go` instead of structured layout |
| `--module` | string | project name | Go module path |
| `--db` | string | `none` | Database backend: `sqlite`, `postgres`, or `none` |
| `--ai` | bool | false | Add OpenAI client wiring |
| `--redis` | bool | false | Add Redis cache wiring |
| `--example` | bool | false | Include example `greet` command and action |
| `--config-scope` | string | `auto` | Config scope: `auto`, `local`, or `global` |
| `--local` | bool | false | Shorthand for `--config-scope local` |
| `--global` | bool | false | Shorthand for `--config-scope global` |
| `--dry-run` | bool | false | Preview operations without writing files |
| `--force` | bool | false | Overwrite existing generated files |
| `--skip-existing` | bool | false | Keep existing files; write only missing ones |
| `--no-manifest` | bool | false | Skip writing `.gokart-manifest.json` for managed scaffolds |
| `--verify` | bool | false | Run `go mod tidy` and `go test ./...` after generation |
| `--verify-only` | bool | false | Run verification only, skip scaffolding |
| `--verify-timeout` | duration | `5m` | Maximum time for verify commands (`0` = no timeout) |
| `--json` | bool | false | Print machine-readable JSON result |

---

## `gokart add`

Adds integrations to an existing structured project without re-scaffolding. Run from the project root directory (where `.gokart-manifest.json` lives).

```bash
gokart add <integration>... [flags]
```

Valid integrations: `sqlite`, `postgres`, `ai`, `redis`. Multiple integrations can be added in a single invocation.

### Examples

```bash
gokart add sqlite
gokart add ai
gokart add redis
gokart add sqlite ai           # add both at once
gokart add postgres --dry-run  # preview changes
gokart add ai --verify         # verify after adding
gokart add sqlite --json       # machine-readable output
```

Destructive override (advanced): `gokart add ai --force` overwrites modified or
untracked generated wiring. Use it only when you intend to discard those local
changes.

### How It Works

`gokart add` re-renders the two files that integration wiring affects:

- `internal/app/context.go`
- `internal/commands/root.go`

It does not re-scaffold the whole project, but dependency and manifest updates
can mutate other files. Specifically:

1. Reads `.gokart-manifest.json` to learn the current project state (module path, existing integrations, config scope).
2. Rejects flat-mode projects — flat projects do not support integrations.
3. Merges the requested integrations with those already enabled.
4. Re-renders only the two integration-affected templates.
5. Checks each file against the manifest hash to detect user modifications.
6. Writes the files (unless `--dry-run`).
7. Runs `go get` for the new dependency packages, then `go mod tidy`.
8. Updates `.gokart-manifest.json` with the new file hashes and integration state (upgrading to v2 format if needed).

Integrations already enabled are skipped with a warning. If all requested integrations are already present, the command exits without writing any files.

### Conflict Detection and `--force`

Before writing, `gokart add` computes the SHA-256 of each existing wiring file
and compares it against the hash stored in the manifest. If the hashes differ,
the file has been modified since it was generated. An existing wiring file
without a matching manifest entry is also a conflict.

| State | Default behavior | With `--force` |
|-------|-----------------|----------------|
| File does not exist | Create | Create |
| File exists, hash matches manifest | Overwrite (safe) | Overwrite |
| File exists, hash differs | Fail with error | Overwrite with warning |
| File exists, not tracked by manifest | Fail with error | Overwrite with warning |

Use `--dry-run` to preview which files would be created or overwritten before committing.

### Dependency Packages

| Integration | Packages fetched via `go get` |
|-------------|-------------------------------|
| `sqlite` | `github.com/dotcommander/gokart/sqlite@latest` |
| `postgres` | `github.com/dotcommander/gokart/postgres@latest`, `github.com/jackc/pgx/v5@latest` |
| `ai` | `github.com/dotcommander/gokart/ai@latest`, `github.com/openai/openai-go/v3@latest` |
| `redis` | `github.com/dotcommander/gokart/cache@latest`, `github.com/redis/go-redis/v9@latest` |

### All `gokart add` Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Preview changes without writing files |
| `--force` | bool | false | Overwrite user-modified files |
| `--verify` | bool | false | Run `go test ./...` after adding integrations |
| `--verify-timeout` | duration | `5m` | Maximum time for verify commands (`0` = no timeout) |
| `--json` | bool | false | Print machine-readable JSON result |

---

## `gokart config`

### `gokart config show`

Prints where gokart stores data on this machine.

```bash
gokart config show
```

Output:

```
Version:     v0.9.0
Config dir:  /Users/you/Library/Application Support
Binary:      /Users/you/go/bin/gokart
```

Useful for debugging when multiple gokart binaries exist or when verifying the installed version matches the expected tag.

---

## Manifest

`gokart new` writes `.gokart-manifest.json` at the project root only for managed scaffolds: global config or integrations. This file is used by `gokart add` to detect user modifications and track which integrations are enabled.

### What the Manifest Tracks

- Generator version that produced the files
- Template root and scaffold mode (`structured` or `flat`)
- Module path and config scope (`use_global`)
- Which integrations are enabled (`sqlite`, `postgres`, `ai`)
- For each generated file: relative path, action taken, SHA-256 of the template content, SHA-256 of the written content, and file mode

### Version 1 vs. Version 2

| Field | v1 | v2 |
|-------|----|----|
| `version` | `1` | `2` |
| `integrations` | absent | present (`{"sqlite":bool,"postgres":bool,"ai":bool}`) |
| `mode` | absent | `"structured"` or `"flat"` |
| `module` | absent | module path string |
| `use_global` | absent | `true` or `false` |

`gokart add` reads both formats. When it updates the manifest it always writes v2. v1 manifests have their integration state inferred from the project's `go.mod`.

### Example Manifest

```json
{
  "version": 2,
  "generator": "gokart/0.9.0",
  "template_root": "templates/structured",
  "existing_file_policy": "fail",
  "generated_at": "2026-03-01T12:00:00Z",
  "mode": "structured",
  "module": "github.com/myorg/mycli",
  "use_global": true,
  "integrations": {
    "sqlite": false,
    "postgres": true,
    "ai": false
  },
  "files": [
    {
      "path": "cmd/main.go",
      "action": "create",
      "template_sha256": "abc123...",
      "content_sha256": "abc123...",
      "mode": 420
    }
  ]
}
```

Plain scaffolds omit the manifest by default. Managed scaffolds write it unless `--no-manifest` is set. Without a manifest, `gokart add` cannot run.

---

## Machine-Readable Output (`--json`)

Both commands accept `--json`. When set, the command writes a single JSON object to stdout and suppresses all human-readable output. The process exits with a structured exit code.

### `gokart new` JSON Output

```json
{
  "outcome": "success",
  "exit_code": 0,
  "preset": "cli",
  "mode": "structured",
  "project_name": "mycli",
  "target_dir": "./mycli",
  "module": "github.com/myorg/mycli",
  "config_scope": "auto",
  "use_global": false,
  "dry_run": false,
  "write_manifest": false,
  "verify_requested": false,
  "verify_only": false,
  "verify_ran": false,
  "verify_passed": false,
  "existing_file_policy": "fail",
  "result": {
    "created": ["cmd/main.go", "go.mod", "..."],
    "overwritten": [],
    "skipped": [],
    "unchanged": []
  },
  "next": {
    "dir": "./mycli",
    "command": "go",
    "args": ["build", "./..."]
  },
  "next_command": "cd './mycli' && go build ./..."
}
```

On failure the object includes `error_code` and `error`:

```json
{
  "outcome": "failure",
  "error_code": "existing_file_conflict",
  "exit_code": 3,
  "conflicts": ["cmd/main.go", "go.mod"],
  "error": "2 destination files already exist (use --force to overwrite or --skip-existing to keep existing files)"
}
```

### `gokart add` JSON Output

```json
{
  "outcome": "success",
  "exit_code": 0,
  "integrations": ["ai"],
  "added": ["ai"],
  "already_present": [],
  "files_created": [],
  "files_overwritten": ["internal/app/context.go", "internal/commands/root.go"],
  "dry_run": false,
  "verify_requested": false,
  "verify_passed": false
}
```

### Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| `0` | success | Operation completed |
| `1` | failure | Unclassified failure |
| `2` | invalid_arguments | Bad flags or argument values |
| `3` | existing_file_conflict | Files exist and policy is `fail` |
| `4` | verify_failed | `go mod tidy` or `go test ./...` failed |
| `5` | target_locked | Another scaffold is running against the target |
| `6` | config_init_failed | Could not initialize config |
| `7` | scaffold_failed | Template rendering or file write failed |
| `8` | json_encode_failed | Could not encode JSON output |
| `9` | manifest_not_found | No `.gokart-manifest.json` (`gokart add` only) |
| `10` | flat_mode_unsupported | Flat project passed to `gokart add` |

---

## Reference

### `gokart new` — Full Flag Table

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--flat` | | false | Single `main.go` layout |
| `--module` | | project name | Go module path |
| `--db` | | `none` | Database backend: `sqlite`, `postgres`, or `none` |
| `--ai` | | false | OpenAI client wiring |
| `--redis` | | false | Redis cache wiring |
| `--example` | | false | Include greet command and action |
| `--config-scope` | | `auto` | `auto`, `local`, or `global` |
| `--local` | | false | Shorthand: `--config-scope local` |
| `--global` | | false | Shorthand: `--config-scope global` |
| `--dry-run` | | false | Preview without writing |
| `--force` | | false | Overwrite existing files |
| `--skip-existing` | | false | Write only missing files |
| `--no-manifest` | | false | Skip `.gokart-manifest.json` for managed scaffolds |
| `--verify` | | false | Run `go mod tidy` + `go test ./...` |
| `--verify-only` | | false | Verify only, no scaffolding |
| `--verify-timeout` | | `5m` | Timeout for verify commands |
| `--json` | | false | Machine-readable JSON output |

### `gokart add` — Full Flag Table

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Preview without writing |
| `--force` | false | Overwrite user-modified files |
| `--verify` | false | Run `go test ./...` after adding |
| `--verify-timeout` | `5m` | Timeout for verify commands |
| `--json` | false | Machine-readable JSON output |

---

## See Also

- [SQLite](/components/sqlite) — SQLite integration added by `--db sqlite`
- [PostgreSQL](/components/postgres) — PostgreSQL integration added by `--db postgres`
- [OpenAI](/components/openai) — OpenAI integration added by `--ai`
- [Migrations](/components/migrate) — Database schema versioning
