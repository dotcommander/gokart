# demo

```bash
go test ./...
go run .
go build -o demo .
```

This is an unmanaged, ordinary Go project. You own every generated file;
`gokart add` will not rewrite it. Keep process exit handling in `main`, write
command-owned output through `kong.Context.Stdout`, and split
business logic into a package when it grows beyond the command handler.

```bash
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o demo .
```
