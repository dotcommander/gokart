# demo

## Build

```bash
go build -o demo .
go install .                              # or: ln -sf "$(pwd)/demo" ~/go/bin/

# Release build with version info
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o demo .
```
