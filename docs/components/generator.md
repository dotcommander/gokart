# Generator — Project Scaffolding

Scaffold new Go CLI projects and add integrations without re-scaffolding.

## Install

Install the tagged CLI:

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.13.0
```

The explicit version keeps installation reproducible.

---

## `gokart new`

Creates a new Go project in a subdirectory of the current working directory. Accepts a project name or a path (e.g., `./myapp`, `/tmp/myapp`). The project name is derived from the final path segment.

```bash
gokart new <project-name> [flags]
gokart init <project-name>              # alias for new
gokart create <project-name>            # alias for new
```

The legacy `gokart new cli <project-name>` form remains accepted for
compatibility, but new scripts and tutorials should use the direct form.

### Basic Examples

```bash
# Flat single-file project (default)
gokart new mycli

# Explicit multi-package project
gokart new mycli --structured

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

### Automatic Layout Selection

**Flat** is the default for plain, `--example`, `--local`, and `--global` scaffolds. It creates a single `main.go`. `--flat` remains available as an explicit assertion.

**Structured** creates a `cmd/` entry point plus `internal/` packages. Select it explicitly with `--structured`, or implicitly by adding SQLite, PostgreSQL, AI, or Redis. Combining `--flat` with `--structured` or any integration is an error; an explicit flat selection is never overridden.

```bash
gokart new mycli                         # flat single-file
gokart new mycli --example               # flat with greet example
gokart new mycli --structured            # structured
gokart new mycli --db sqlite             # structured automatically
gokart new mycli --structured --global   # managed and compatible with add
```

Configuration scope does not choose the layout. Every flat scaffold is
unmanaged and manifest-free, including `--global`. Every structured scaffold is
managed by default. Use `--no-manifest` only to opt a structured scaffold out of
management; in flat mode it produces an "already unmanaged" warning.

### Generated File Tree

#### Structured (`--structured` or any integration)

```
mycli/
├── .gokart-manifest.json              # Managed generated-file hashes
├── .gitignore
├── README.md                          # Build commands
├── cmd/
│   └── main.go                        # Entry point
├── internal/
│   └── commands/
│       └── root.go                    # Kong command tree and dependency binding
└── go.mod
```

Structured manifests let `gokart add` safely patch generated wiring later.
`--global` adds config bootstrap code. Integration flags add
`internal/app/context.go`. `--no-manifest` omits only the management metadata;
it does not change the selected layout.

With `--example`:

```
internal/
├── commands/
│   └── greet.go                       # Example command
└── actions/
    ├── greet.go                       # Business logic (testable)
    └── greet_test.go                  # Example test
```

#### Flat (default)

```
mycli/
├── .gitignore
├── README.md
├── go.mod
├── main.go
└── main_test.go                       # With --example
```

Generated modules declare exactly `go 1.26.0` and do not emit a `toolchain`
directive.

### Integration Flags

These flags wire a data store or API client into the project at generation time. They add the import, the dependency to `go.mod`, and the wiring in `internal/app/context.go` and `internal/commands/root.go`.

| Flag | Package added | What it wires |
|------|---------------|---------------|
| `--db sqlite` | `github.com/dotcommander/gokart/sqlite` | `*sql.DB` via `sqlite.Open` |
| `--db postgres` | `github.com/dotcommander/gokart/postgres`, `github.com/jackc/pgx/v5` | `*pgxpool.Pool` via `postgres.Open` |
| `--ai` | `github.com/openai/openai-go/v3` | `openai.Client` via `openai.NewClient` |
| `--redis` | `github.com/dotcommander/gokart/cache`, `github.com/redis/go-redis/v9` | `*cache.Cache` via `cache.Open` |

Flags may be combined:

```bash
gokart new mycli --db postgres --ai
gokart new mycli --db sqlite --example
gokart new mycli --redis
gokart new mycli --redis --db postgres
```

Generated integrations are command-lazy. Kong constructs `app.Dependencies`
through `app.New` only when the selected command requests it. `app.Context`
remains an alias for compatibility. Help, no-argument usage, version output, and
`greet` do not connect to PostgreSQL or Redis, create a SQLite file, or require
an API credential. Cleanup runs only after construction and joins database or
cache close errors with the command result.

### Config Scope

Controls whether the generated project bootstraps `config.yaml` under the platform user config directory returned by `os.UserConfigDir()`.

```bash
--config-scope auto     # default: local-only
--config-scope local    # no ~/.config/<app>/ bootstrap
--config-scope global   # always bootstrap ~/.config/<app>/config.yaml
```

The shorthand flags `--local` and `--global` are aliases for `--config-scope local` and `--config-scope global`. They cannot be used together, and cannot be combined with `--config-scope`.

The default scope is `local` in both structured and flat mode. Use `--global` when you want generated config-dir bootstrap code. Scope never forces structured mode.

### Manifest

Flat scaffolds never write `.gokart-manifest.json`. Structured scaffolds always
write it unless `--no-manifest` is set. Without a manifest, `gokart add` cannot
safely update the project. There is no automatic flat-to-structured conversion
or manifest schema migration during `new`.

### Conflict Handling

When the target directory already exists and contains files, `gokart new`
checks every destination. By default it writes nothing and reports all
conflicting paths.

The default is to fail without writing and report the conflicts. Use
`--force` to overwrite them or `--skip-existing` to keep existing files and
write only missing ones. The two flags cannot be combined.

### Verification

Dependency preparation is mandatory for every new scaffold: GoKart runs the
pinned `go get` operation and `go mod tidy` once, even with `--no-verify`. A
failure keeps the generated files, exits with scaffold-failure code `7`, and
prints the exact recovery command. It never reports success for a broken module.

Unless disabled, every scaffold then runs `go test ./...` and `go build ./...`.
Generated PostgreSQL and Redis tests and the `greet` example do not connect to
services, so these integrations have no verification exemption.
`GOKART_AUTO_VERIFY=0` disables the tests and build for pipelines that verify
separately; dependency preparation still runs.

`--verify-only` cannot be combined with `--dry-run`. Generation flags (`--flat`, `--structured`, `--db`, `--ai`, `--redis`, `--example`, `--config-scope`, `--force`, `--skip-existing`, `--no-manifest`) are ignored when `--verify-only` is set.

Dry-run dependency preparation, tests, and build happen in a temporary
scaffold, which is then removed. No files are written to the target.

Successful human output stays concise; detailed file actions remain in dry-run
and JSON output:

```text
Created tvguide (flat)
Verified: tests and build

Next:
  cd 'tvguide'
  go run . greet --name World
  go build -o tvguide .
```

### All `gokart new` Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--flat` | bool | false | Use a single-package root layout (the default) |
| `--structured` | bool | false | Use a managed `cmd/` and `internal/` layout for later integrations |
| `--module` | string | project name | Go module path |
| `--db` | string | `none` | Database backend: `sqlite`, `postgres`, or `none` |
| `--ai` | bool | false | Add OpenAI client wiring |
| `--redis` | bool | false | Add Redis cache wiring |
| `--example` | bool | false | Include a runnable `greet` command and tests |
| `--config-scope` | string | `auto` | Config scope: `auto`, `local`, or `global` |
| `--local` | bool | false | Shorthand for `--config-scope local` |
| `--global` | bool | false | Shorthand for `--config-scope global` |
| `--dry-run` | bool | false | Preview operations without writing files |
| `--force` | bool | false | Overwrite existing generated files |
| `--skip-existing` | bool | false | Keep existing files; write only missing ones |
| `--no-manifest` | bool | false | Opt a structured project out of GoKart management |
| `--verify` | bool | default-on | Explicitly request post-generation tests and build |
| `--no-verify` | bool | false | Skip tests and build, but still prepare dependencies |
| `--verify-only` | bool | false | Tidy, test, and build an existing project without scaffolding |
| `--verify-timeout` | duration | `5m` | Maximum time for verify commands (`0` = no timeout) |
| `--json` | bool | false | Print machine-readable JSON result |

---

## `gokart add`

Adds integrations to an existing managed structured project without
re-scaffolding. Run from the project root directory, where
`.gokart-manifest.json` lives.

For a flat project, add the relevant GoKart package manually. Start future
projects that need generator-managed growth with
`gokart new <name> --structured`.

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
| `sqlite` | `github.com/dotcommander/gokart/sqlite@v0.13.0` |
| `postgres` | `github.com/dotcommander/gokart/postgres@v0.13.0`, `github.com/jackc/pgx/v5@v5.10.0`, `github.com/pressly/goose/v3@v3.27.2` |
| `ai` | `github.com/openai/openai-go/v3@v3.42.0` |
| `redis` | `github.com/dotcommander/gokart/cache@v0.13.0`, `github.com/redis/go-redis/v9@v9.21.0` |

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
Version:     v0.13.0
Config dir:  /Users/you/Library/Application Support
Binary:      /Users/you/go/bin/gokart
```

Useful for debugging when multiple gokart binaries exist or when verifying the installed version matches the expected tag.

---

## Manifest

`gokart new` writes `.gokart-manifest.json` at the root of every structured
scaffold unless `--no-manifest` is set. Flat projects never receive a manifest.
The file lets `gokart add` detect user modifications and track integrations.

### What the Manifest Tracks

- Generator version that produced the files
- Template root and scaffold mode (`structured` or `flat`)
- Module path and config scope (`use_global`)
- Which integrations are enabled (`sqlite`, `postgres`, `ai`, `redis`)
- For each generated file: relative path, action taken, SHA-256 of the template content, SHA-256 of the written content, and file mode

### Version 1 vs. Version 2

| Field | v1 | v2 |
|-------|----|----|
| `version` | `1` | `2` |
| `integrations` | absent | present (`{"sqlite":bool,"postgres":bool,"ai":bool,"redis":bool}`) |
| `mode` | absent | `"structured"` or `"flat"` |
| `module` | absent | module path string |
| `use_global` | absent | `true` or `false` |

`gokart add` reads both formats. When it updates the manifest it always writes v2. v1 manifests have their integration state inferred from the project's `go.mod`.

### Example Manifest

```json
{
  "version": 2,
  "generator": "gokart",
  "generator_version": "v0.13.0",
  "template_root": "templates/structured",
  "existing_file_policy": "fail",
  "mode": "structured",
  "module": "github.com/myorg/mycli",
  "use_global": true,
  "integrations": {
    "sqlite": false,
    "postgres": true,
    "ai": false,
    "redis": true
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

Flat scaffolds omit the manifest. Structured scaffolds write it by default.
Without a structured manifest, `gokart add` cannot run.

---

## Machine-Readable Output (`--json`)

Both commands accept `--json`. When set, the command writes a single JSON object to stdout and suppresses all human-readable output. The process exits with a structured exit code.

### `gokart new` JSON Output

Example for `gokart new /work/mycli --module github.com/myorg/mycli --no-verify --json`:

```json
{
  "outcome": "success",
  "exit_code": 0,
  "preset": "cli",
  "mode": "flat",
  "project_name": "mycli",
  "target_dir": "/work/mycli",
  "module": "github.com/myorg/mycli",
  "config_scope": "auto",
  "use_global": false,
  "dry_run": false,
  "write_manifest": false,
  "verify_requested": false,
  "verify_only": false,
  "verify_ran": false,
  "verify_passed": false,
  "checks": [
    {"command": "go get github.com/alecthomas/kong@v1.15.0", "status": "passed"},
    {"command": "go mod tidy", "status": "passed"}
  ],
  "existing_file_policy": "fail",
  "result": {
    "created": [".gitignore", "README.md", "go.mod", "main.go"],
    "overwritten": [],
    "skipped": [],
    "unchanged": []
  },
  "next": {
    "dir": "/work/mycli",
    "command": "go",
    "args": ["build", "-o", "mycli", "."]
  },
  "next_command": "cd '/work/mycli' && go build -o mycli .",
  "next_steps": [
    "cd '/work/mycli'",
    "go build -o mycli ."
  ]
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
| `4` | verify_failed | Generated-project tests or build failed |
| `5` | target_locked | Another scaffold is running against the target |
| `6` | config_init_failed | Could not initialize config |
| `7` | scaffold_failed | Scaffolding or dependency preparation failed |
| `8` | json_encode_failed | Could not encode JSON output |
| `9` | manifest_not_found | No `.gokart-manifest.json` (`gokart add` only) |
| `10` | flat_mode_unsupported | Flat project passed to `gokart add` |

---

## See Also

- [SQLite](sqlite.md) — SQLite integration added by `--db sqlite`
- [PostgreSQL](postgres.md) — PostgreSQL integration added by `--db postgres`
- [openai-go](https://github.com/openai/openai-go) — official SDK used by `--ai`
- [Migrations](migrate.md) — Database schema versioning
