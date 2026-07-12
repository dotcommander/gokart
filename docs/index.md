# GoKart documentation

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.11.0
gokart new mycli --db sqlite --example
```

GoKart supplies small infrastructure modules and a generator, while the generated application remains ordinary Go that you own.

## Newcomer path

1. [Why GoKart?](why-gokart.md) — decide whether to use GoKart, copy its generated patterns, or use upstream packages directly.
2. [Build a SQLite CLI](getting-started.md) — scaffold, configure, migrate, write, query, test, and build an offline application.
3. [Own the generated code](generated-code.md) — understand rewrite boundaries, manifests, conflicts, removal, and escape hatches.
4. [Recipes](recipes.md) — copy focused application patterns.
5. [Testing](testing.md) — test commands, actions, SQLite, HTTP, and service integrations.
6. [Full-service example](examples/README.md#service-composition) — expand the same model to PostgreSQL, Redis, and HTTP.

The canonical maintainer boundary is [GoKart philosophy](../PHILOSOPHY.md). These guides translate it into application-development choices.

## Reference

- [Generator](components/generator.md), [CLI](api/cli.md), and [configuration and state](api/gokart.md)
- [Logger](components/logger.md)
- [SQLite](components/sqlite.md), [PostgreSQL](components/postgres.md), and [migrations](components/migrate.md)
- [Redis cache](components/cache.md)
- [Web](components/web.md), [responses](components/response.md), and [validation](components/validate.md)
- [Runnable examples](examples/README.md)
