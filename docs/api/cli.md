# Build a CLI

```go
app := cli.NewApp("myapp", "1.0.0").
    WithDescription("Manage my application").
    WithEnvPrefix("MYAPP").
    WithStandardFlags()

app.AddCommand(cli.Command("run", "Run the application", func(cmd *cobra.Command, args []string) error {
    return run(cmd.Context())
}))

if err := app.Run(); err != nil {
    os.Exit(1)
}
```

The `cli` module provides small Cobra and Lip Gloss helpers. Your `main` function owns exit status. Commands return errors.

## Install

```bash
go get github.com/dotcommander/gokart/cli@v0.11.0
```

## Configure the application

`NewApp(name, version)` returns an `*App`. Chain only the setup you need:

| Method | Behavior |
|---|---|
| `WithDescription(text)` | Sets the root command's short description. |
| `WithLongDescription(text)` | Sets detailed root help. |
| `WithConfig(path)` | Reads one explicit config file. |
| `WithConfigName(name)` | Searches `.` and the platform app config directory for YAML, then `/etc/<app>`. |
| `WithEnvPrefix(prefix)` | Enables Viper environment loading and maps `.` and `-` to `_`. |
| `WithStandardFlags()` | Adds `--config`, `--verbose`/`-v`, and `--quiet`/`-q`. |
| `Root()` | Returns the real `*cobra.Command`. |
| `Viper()` | Returns the real `*viper.Viper`. |
| `RunWithArgs(args)` | Executes explicit arguments in tests. |

A missing searched config file is allowed. An explicit config file that exists but cannot be read or parsed returns an error.

## Add commands

```go
serve := cli.Command("serve", "Start the server", func(cmd *cobra.Command, args []string) error {
    return server.Serve(cmd.Context())
})

show := cli.CommandWithArgs("show <id>", "Show one record", 1, func(cmd *cobra.Command, args []string) error {
    return showRecord(cmd.Context(), args[0])
})

admin := cli.Group("admin", "Administrative commands")
admin.AddCommand(show)
app.AddCommand(serve).AddCommand(admin)
```

These helpers return ordinary Cobra commands. Use Cobra directly when you need custom argument validation, completion, flags, or lifecycle hooks.

## Write styled output

```go
cli.Success("saved %s", path) // stdout
cli.Info("reading configuration")
cli.Warning("using fallback")
cli.Error("save failed: %v", err) // stderr
```

`Success`, `Info`, `Warning`, `Dim`, and `Bold` write to the process stdout. `Error` writes to process stderr. They do not expose mutable package-level writer overrides.

For command output that must be captured or redirected, write through Cobra:

```go
fmt.Fprintln(cmd.OutOrStdout(), result)
fmt.Fprintln(cmd.ErrOrStderr(), warning)
```

Tests can then use `cmd.SetOut` and `cmd.SetErr`.

## Show progress

```go
spinner := cli.NewSpinner("Loading").WithWriter(cmd.OutOrStdout())
spinner.StartWithContext(cmd.Context())
defer spinner.Stop()

progress := cli.NewProgress("Importing", len(files)).SetWriter(cmd.OutOrStdout())
for range files {
    progress.Increment()
}
progress.Done()
```

Spinners animate only on a terminal. In non-TTY output they print the message once. `WithFrames` and `WithDelay` customize animation. `Update`, `StopWithMessage`, `StopSuccess`, and `StopError` control the final state.

`WithSpinner(message, fn)` is the compact process-stream helper when you do not need an injected writer.

## Render tables and lists

```go
table := cli.NewTable("NAME", "STATUS").
    AddRow("api", "ready").
    AddRow("worker", "stopped").
    SetWriter(cmd.OutOrStdout())
table.Print()
```

`Table.String` returns rendered text. `SimpleTable`, `KeyValue`, `List`, and `NumberedList` are process-stdout shortcuts.

## Open an editor

```go
text, err := cli.CaptureInput("initial text\n", ".md")
```

`CaptureInput` uses the configured editor. `CaptureInputWithEditor` accepts an explicit editor command for deterministic callers and tests.

## Style help

```go
cli.SetStyledHelp(app.Root())
```

This changes Cobra's help and usage templates. The root `App` already uses the package's normal Cobra ownership; call this only when you want the styled templates.

## Migration from v0.10

| Removed API | Replacement |
|---|---|
| `SetOutput`, `SetErrOutput`, `Output`, `ErrOutput` | Cobra `SetOut`, `SetErr`, `OutOrStdout`, and `ErrOrStderr` |
| `Fatal`, `FatalErr`, `Must` | Return errors; let `main` choose `os.Exit` |

## See also

- [Generator](../components/generator.md)
- [Root package](gokart.md)
- [Project philosophy](../../PHILOSOPHY.md)
