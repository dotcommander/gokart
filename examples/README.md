# Examples

Complete application examples demonstrating GoKart patterns.

## Full Applications

| Example | Description |
|---------|-------------|
| [http-service/](http-service/) | Minimal HTTP API with chi router, response helpers, and graceful shutdown |
| [cli-app/](cli-app/) | Deterministic TV guide with injected arguments and writers |

Run either standalone module with workspace replacements disabled:

```bash
cd examples/cli-app && GOWORK=off go run . now --channel WGBH
cd examples/http-service && GOWORK=off go run .
```

## Component Examples

For focused single-component examples, see [docs/examples/](../docs/examples/):

- [cache](../docs/examples/cache/) — Redis caching with Remember pattern **(needs Redis)**
- [cli](../docs/examples/cli/) — Cobra app for ecosystem-style CLIs
- [config](../docs/examples/config/) — Typed configuration with environment binding
- [fslock](../docs/examples/fslock/) — Nonblocking advisory file lock
- [full-service](../docs/examples/full-service/) — Combined PostgreSQL + cache + web service **(needs PostgreSQL + Redis)**
- [generator](../docs/examples/generator/) — Copy-paste `gokart new` / `add` / `--verify-only` flow
- [http-server](../docs/examples/http-server/) — HTTP router, bounded binding, and validation
- [kongapp](../docs/examples/kongapp/) — Caller-owned Kong tree with Initialize/cleanup
- [logger](../docs/examples/logger/) — Structured logging and file logger
- [migrate](../docs/examples/migrate/) — Goose migrations against a temp directory
- [postgres](../docs/examples/postgres/) — PostgreSQL connection and transactions **(needs PostgreSQL)**
- [sqlite](../docs/examples/sqlite/) — SQLite database operations
- [state](../docs/examples/state/) — Small CLI state under the platform config dir
- [upstream-direct](../docs/examples/upstream-direct/) — Same service with no GoKart imports (delete-the-wrapper path)
- [validate](../docs/examples/validate/) — Bounded binding and validation without opening a port
