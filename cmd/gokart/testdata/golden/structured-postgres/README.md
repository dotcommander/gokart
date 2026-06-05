# demo

## Build

```bash
go build -o demo ./cmd
go install ./cmd                          # or: ln -sf "$(pwd)/demo" ~/go/bin/

# Release build with version info
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o demo ./cmd
```

## Structure

```
cmd/main.go             Entry point
internal/commands/      CLI commands (Cobra)
internal/actions/       Business logic
internal/app/           App config/context
```

## Configuration

Set `DATABASE_URL` environment variable for PostgreSQL.
