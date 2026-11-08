# Generator example

The generator is exercised by the tutorials, not a Go file — this README is the copy-paste path:

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.16.0

# Preview first; nothing is written
gokart new notes --db sqlite --example --dry-run

# Flat single-file CLI (default), verified
gokart new tvguide --example
cd tvguide && go run . greet --name World

# Structured project that stays managed for later `gokart add`
gokart new service --structured --global
cd service && gokart add postgres redis --dry-run

# Existing project: tidy + test + build without scaffolding
gokart new "$PWD" --verify-only
```

Then continue with the [TV-guide tutorial](../../getting-started.md), which turns the flat `greet` example into a tested, filterable CLI, and [generated-code ownership](../../generated-code.md) before using `gokart add` on a project you have customized.
