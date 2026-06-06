package main

const logo = `
   ____       _  __          _
  / ___| ___ | |/ /__ _ _ __| |_
 | |  _ / _ \| ' // _' | '__| __|
 | |_| | (_) | . \ (_| | |  | |_
  \____|\___/|_|\_\__,_|_|   \__|
`

const rootLongDescription = logo + `
	gokart new <name> [flags]
	gokart new cli <name> [flags]
	gokart add <integration>... [flags]

  --db             Database backend: sqlite|postgres|none
  --ai             OpenAI client (openai-go/v3)
  --redis          Redis cache (go-redis/v9)
  --example        Include example greet command/action scaffold
  --flat           Single main.go (no internal/)
  --local          Local-only config (default)
  --global         Global config
  --config-scope   Config scope: auto|local|global
  --module         Custom module path
  --dry-run        Preview scaffold operations without writing files
  --force          Overwrite ALL existing files (including user edits)
  --skip-existing  Skip files that already exist; only write new/missing ones
  --no-manifest    Skip managed .gokart-manifest.json
  --verify         Run go mod tidy and go test ./... after generation
  --verify-only    Run verification only against an existing project directory
  --verify-timeout Max duration for --verify commands (default 5m, 0 disables)
  --json           Print machine-readable JSON result`

const newCommandLong = `Create a new Go project with sensible defaults and optional integrations.

Structured mode (default) creates:
  myapp/
  ├── cmd/main.go                # Entry point
  ├── internal/commands/         # Cobra command definitions
  └── go.mod

Use --example to include greet command/action examples.

Flat mode creates a single main.go for quick scripts.`

const newCommandExample = `  # Basic structured project
  gokart new myapi

  # Explicit preset (same output as command above)
  gokart new cli myapi

  # With PostgreSQL and OpenAI
  gokart new myapi --db postgres --ai

  # With Redis cache
  gokart new myapi --redis

  # Include example command/action scaffold
  gokart new myapi --example

  # With SQLite for local-first CLI
  gokart new mycli --db sqlite

  # Quick script (single main.go)
  gokart new script --flat

  # Preview without writing files
  gokart new myapi --dry-run

  # Overwrite existing generated files
  gokart new myapi --force

  # Generate and verify immediately
  gokart new myapi --verify

  # Verify an existing generated project without changing files
  gokart new myapi --verify-only

  # JSON output for CI tooling
  gokart new myapi --dry-run --json

  # Custom module path
  gokart new myapi --module github.com/myorg/myapi`

const rootHelpTemplate = `{{.Long}}

  gokart new myapp
  gokart new cli myapp
  gokart init myapp              (alias for new)
  gokart new myapp --db postgres --ai
  gokart new myapp --redis
  gokart add sqlite ai

  Defaults: local config · manifest only for global config/integrations

  gokart new --help    Full options
  gokart add --help    Add integrations
`
