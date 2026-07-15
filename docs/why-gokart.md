# Why GoKart?

```bash
gokart new notes --db sqlite --example --dry-run
```

Use this preview when you want a typed Kong application with explicit dependency wiring, an action boundary, SQLite defaults, and verification without designing the first directory layout yourself. Integrations select the structured layout automatically; a plain scaffold defaults to one flat `main.go`.

## Decide quickly

Use GoKart when you want recurring infrastructure setup to be consistent but still want to edit normal Go. Its modules add bounded policy such as SQLite pragmas, PostgreSQL pool defaults, typed configuration, atomic state, Redis key prefixes, or graceful HTTP shutdown.

Use Kong or another parser plus upstream packages directly when their APIs already express everything you need, or your organization already owns equivalent startup and lifecycle conventions. GoKart is not an application framework, ORM, dependency-injection container, or domain layer.

| Approach | Best fit | Trade-off |
|---|---|---|
| Generate and keep GoKart modules | You value the defaults and escape hatches | Your app imports the selected small modules |
| Generate, then copy the patterns | You want a reviewed layout but no generator lifecycle | You own future wiring and upgrades |
| Start with upstream packages | You already know the exact architecture and policies | You write lifecycle and verification setup |

Kong provides command parsing, typed commands, dependency binding through `Context.Run`, and command-scoped writers. It does not decide how to load configuration, open databases, close clients, organize testable actions, or scaffold those choices. GoKart writes one concrete ownership model. The underlying `*sql.DB`, `*pgxpool.Pool`, and Redis client remain reachable.

## Do not use GoKart when

- You need application or domain policy generated for you.
- Your platform already owns configuration, logging, database, and shutdown conventions.
- You want generated files to remain opaque or framework-owned.
- You need a wrapper that mirrors every upstream method; call upstream directly.
- You cannot accept pre-1.0 change; pin and review migration guidance, or stay upstream.

Next, build the [SQLite CLI tutorial](getting-started.md), then read [generated-code ownership](generated-code.md) before using `gokart add`.
