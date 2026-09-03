# kongapp

`kongapp` provides the repeated execution shell around a consumer-owned
[Kong](https://github.com/alecthomas/kong) command tree. It binds caller-owned
context and streams, handles help without terminating an embedding process, and
supports explicit post-parse initialization and teardown.

Applications retain ownership of command structs, flags, configuration,
dependencies, signals, domain behavior, and exit policy:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

err := kongapp.Execute(ctx, os.Args[1:], kongapp.Streams{
    In: os.Stdin, Out: os.Stdout, Err: os.Stderr,
}, &commands, kongapp.Config[commandTree]{
    Name: "example",
    Options: []kong.Option{kong.Bind(service)},
})
```

Use `Config.Initialize` only for work that must happen after successful command
selection. Help and version paths never call it. Return a cleanup function when
initialization acquires resources.
