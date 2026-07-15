# Own the generated code

```bash
gokart new notes --db sqlite --example
cd notes
gokart add redis --dry-run
```

Everything in `notes` belongs to you. The manifest only records which generated files are safe for `gokart add` to rewrite.

## Ownership map

| Path | Owner | Generator behavior |
|---|---|---|
| `cmd/main.go` | You | Created once; never rewritten by `add` |
| `internal/actions/**` | You | Put business behavior here |
| command files except `root.go` | You | Put commands and flags here |
| `internal/app/context.go` | Shared rewrite boundary | `add` may re-render it |
| `internal/commands/root.go` | Shared rewrite boundary | `add` may re-render it |
| `internal/app/config.go` | You | Generated for global config, not rewritten by `add` |
| `go.mod`, `go.sum` | Go tooling and you | `add` runs `go get` and `go mod tidy` |
| `.gokart-manifest.json` | Generator metadata | Records hashes and integration state |
| `README.md`, `.gitignore` | You | Generated starting points |

New command, action, service, and repository files are safe extension seams. If you customize `context.go` or `root.go`, finish adding integrations first or stop using `gokart add`.

## Manifests and conflicts

Managed scaffolds record SHA-256 hashes. Before replacing the two wiring files, `gokart add` compares them with their recorded hashes. A matching file is safe to rewrite; a changed or untracked rewrite target is a conflict.

- `--dry-run` previews changes.
- `--force` discards conflicting edits in rewrite targets; copy or commit your edits first.
- `--no-manifest` suppresses the manifest and therefore prevents later `gokart add` use.

`gokart add` supports managed structured projects only. It re-renders the two wiring files, fetches dependencies, tidies the module, and updates the manifest. It does not regenerate the project.

Plain and global scaffolds default to flat and never write a manifest. Start with
`gokart new <name> --structured --global` when you want global configuration
plus later generator-managed integration updates.

## Stop or remove

To stop using the generator, finish the desired integrations, delete `.gokart-manifest.json`, and maintain the ordinary source yourself. The application does not call the generator at runtime.

There is no `gokart remove`. To remove an integration, delete its fields, imports, config keys, initialization, cleanup, and consumers; run `go mod tidy`; delete the now-inaccurate manifest; then run `go test ./...`.

## Replace a wrapper

| GoKart surface | Direct replacement |
|---|---|
| generated Kong command structs | Kong directly; the generated application already uses typed commands, `Context.Run` binding, and `kong.Context` writers |
| `sqlite` | `database/sql` with `modernc.org/sqlite`; own pragmas and transaction cleanup |
| `postgres` | `pgxpool.NewWithConfig` and `pgx.BeginFunc` or an explicit transaction |
| `cache.Cache` | `redis.NewClient` or `redis.ParseURL`; apply prefixes locally |
| generated OpenAI client | `openai.NewClient` from `github.com/openai/openai-go/v3` |
| `logger.New` | `slog.New` with a text or JSON handler |
| `web` | `net/http`, chi, `encoding/json`, and validator directly |
| `migrate` | `goose.Provider` from `pressly/goose/v3` |
| config/state helpers | Viper or `encoding/json`, `os.UserConfigDir`, and atomic writes |

Replace one boundary at a time. The [recipes](recipes.md) show the public escape hatches.
