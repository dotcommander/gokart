# Run a Kong command tree with caller-owned streams

Use `kongapp` when you want the repeated parse-and-run shell around a Kong command tree while keeping configuration, dependencies, signal handling, and the exit policy in your application.

## Install

```bash
go get github.com/dotcommander/gokart/kongapp@v0.17.0
```

Tagged with the workspace `v0.17.0` release.

## First action: parse, initialize, run, clean up

Prerequisite: a Kong command tree whose `Run` methods accept the dependencies you bind. `config` examples below assume module `example.com/app` and the `kongapp` package on the module path.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

err := kongapp.Execute(ctx, os.Args[1:], kongapp.Streams{
	In:  os.Stdin,
	Out: os.Stdout,
	Err: os.Stderr,
}, &commands, kongapp.Config[commandTree]{
	Name:        "example",
	Description: "Example application",
	Options:     []kong.Option{kong.Bind(service)},
	Initialize: func(ctx context.Context, parsed *kong.Context, tree *commandTree, args []string) (kongapp.Cleanup, error) {
		// Runs only after a command parses successfully.
		// Return a cleanup function to release what initialization acquires.
		return func() error { return nil }, nil
	},
})
if err != nil {
	// Your process decides whether this becomes os.Exit(1).
	return err
}
```

Expected observable result: the selected command writes to `Streams.Out`, diagnostics and Kong errors go to `Streams.Err`, and `Execute` returns the command error joined with any cleanup error (checkable with `errors.Is` for each). This closed loop is source-checked against `kongapp/app_test.go` (`TestExecuteRunsLifecycleWithCallerOwnedDependencies`, `TestExecuteJoinsCommandAndCleanupFailures`); it was not executed as a standalone program here.

## How it works

`Execute` builds a Kong parser with `kong.Name`, `kong.Writers(streams.Out, streams.Err)`, `kong.BindTo(ctx, (*context.Context)(nil))`, and `kong.Bind(streams)`, then appends your `Config.Options`. Command `Run` methods can therefore request `context.Context`, `kongapp.Streams`, and any value you bound. `Config.Vars`, `Config.Help`, and `Config.DefaultArgs` map to the matching Kong options; when `args` is empty and `DefaultArgs` is set, the defaults are used without mutating the config slice.

`Config.Initialize` runs after successful parsing and before the selected command, receiving the parsed command, the populated tree, and the effective arguments. Help and version paths return before initialization, so help output never initializes application resources.

## Failure behavior

| Symptom | Cause | Fix |
|---|---|---|
| `kongapp: context is required` | nil context passed | Pass a non-nil context; own cancellation with `signal.NotifyContext` |
| `kongapp: command tree is required` | nil tree pointer | Pass `&commands` for your command struct |
| `kongapp: application name is required` | empty `Config.Name` | Set `Name` to the program name shown in help |
| `kongapp: stdout is required` / `stderr is required` | nil `Streams.Out` / `Streams.Err` | Always supply both writers, even in tests (use `bytes.Buffer`) |
| Initialize returned an error | Setup failed after parsing | Fix the setup; the command does not run |

The parser's `Exit` is intercepted so help and version never terminate an embedding process; `Execute` returns nil for those paths.

## Next safe variations

- Join command and cleanup errors without losing either: return the `Execute` error and inspect it with `errors.Is`.
- Test with explicit arguments and buffers exactly as `kongapp/app_test.go` does; assert on `Streams.Out` content and initialization order instead of replacing process streams.
- For styled Cobra applications instead of Kong, see the [CLI reference](../api/cli.md).

## See also

- [CLI (Cobra)](../api/cli.md) — the ecosystem-style alternative; `kongapp` is the Kong counterpart
- [Root configuration](../api/gokart.md) — typed config and state helpers commands often consume
