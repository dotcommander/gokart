# Examples

Complete application examples demonstrating GoKart patterns.

## Full Applications

| Example | Description |
|---------|-------------|
| [http-service/](http-service/) | Minimal HTTP API with chi router, response helpers, and graceful shutdown |
| [cli-app/](cli-app/) | Focused Kong CLI with typed commands and command-scoped output writers |

Run either standalone module with workspace replacements disabled:

```bash
cd examples/cli-app && GOWORK=off go run .
cd examples/http-service && GOWORK=off go run .
```

## Component Examples

For focused single-component examples, see [docs/examples/](../docs/examples/):

- [cache](../docs/examples/cache/) — Redis caching with Remember pattern
- [config](../docs/examples/config/) — Typed configuration with environment binding
- [full-service](../docs/examples/full-service/) — Combined PostgreSQL + cache + web service
- [http-server](../docs/examples/http-server/) — HTTP router and server setup
- [logger](../docs/examples/logger/) — Structured logging and file logger
- [postgres](../docs/examples/postgres/) — PostgreSQL connection and transactions
- [sqlite](../docs/examples/sqlite/) — SQLite database operations
