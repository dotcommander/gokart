# Load configuration and state

```go
type Config struct {
    Address string `config:"address" default:":8080"`
    Token   string `config:"token" required:"true"`
}

cfg, err := gokart.ParseConfig[Config](map[string]any{
    "token": os.Getenv("APP_TOKEN"),
})
```

The root module owns typed configuration parsing, platform configuration directories, and small JSON state files.

## Install

```bash
go get github.com/dotcommander/gokart@v0.11.0
```

## Parse a configuration map

`ParseConfig[T](map[string]any)` converts a map into a struct. `T` must be a struct.

```go
type Config struct {
    Host    string  `config:"host" default:"localhost"`
    Port    int     `config:"port" default:"8080"`
    Enabled bool    `config:"enabled" required:"true"`
}

cfg, err := gokart.ParseConfig[Config](map[string]any{"enabled": true})
```

- `config:"name"` selects the input key; the default is the lowercase field name.
- `default:"value"` supports strings, integers, floats, and booleans.
- `required:"true"` requires a supplied value or a default.
- Anonymous struct fields are flattened.
- Numeric conversions are allowed. String-to-number, string-to-bool, and bool-to-other conversions are rejected.

`MustParseConfig` panics when invalid configuration represents a programmer error during initialization. Prefer `ParseConfig` at normal runtime boundaries.

## Load files and environment variables

```go
type FileConfig struct {
    Database struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"database"`
}

cfg, err := gokart.LoadConfig[FileConfig]("config.yaml", "config.json")
```

`LoadConfig` reads the first existing path. It returns an error when paths were supplied but none exist. Viper also reads environment variables and maps dots to underscores.

Use `LoadConfigWithDefaults(defaults, paths...)` to preserve caller-supplied defaults before file and environment overrides:

```go
cfg, err := gokart.LoadConfigWithDefaults(
    FileConfig{Database: struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    }{Host: "localhost", Port: 5432}},
    "config.yaml",
)
```

## Initialize an application config directory

```go
dir, err := gokart.ConfigDir("myapp")
err = gokart.EnsureConfigDir("myapp", []byte("address: :8080\n"))
```

`ConfigDir` returns `<os.UserConfigDir()>/myapp` and creates it with mode `0755`. `EnsureConfigDir` race-safely creates `config.yaml` with mode `0644` only when it does not already exist.

## Save application state

```go
type State struct {
    LastProject string `json:"last_project"`
}

if err := gokart.SaveState("myapp", "state.json", State{LastProject: "demo"}); err != nil {
    return err
}

state, err := gokart.LoadState[State]("myapp", "state.json")
if errors.Is(err, os.ErrNotExist) {
    state = State{}
}
```

`SaveState` atomically publishes indented JSON with mode `0600`. `LoadState` returns `os.ErrNotExist` for a missing file. `StatePath(app, filename)` returns the platform path, or an empty string when the user config directory cannot be determined.

State files are for small CLI state. Use a database or purpose-built store for concurrent records, queries, or large data.

## Modules

| Module | Purpose |
|---|---|
| [`gokart/cli`](cli.md) | Cobra application construction and terminal presentation |
| [`gokart/cache`](../components/cache.md) | Redis construction, prefixes, JSON, and Remember |
| [`gokart/web`](../components/web.md) | HTTP setup, bounded binding, responses, and validation |
| [`gokart/postgres`](../components/postgres.md) | pgx pool setup and transactions |
| [`gokart/sqlite`](../components/sqlite.md) | pure-Go SQLite setup and operations |
| [`gokart/migrate`](../components/migrate.md) | goose migrations |
| [`gokart/logger`](../components/logger.md) | slog setup |

The removed `gokart/ai` and `gokart/fs` modules have no forwarding packages. See the [v0.11 migration table](../../README.md#v011-migration).
