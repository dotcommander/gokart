1. Startup (`cmd/gokart/main.go`)
   1.1 `main()` bootstraps the CLI app.
       1.1.1 Calls `cli.NewApp("gokart", "0.1.0")`.
       1.1.2 Adds short and long descriptions (logo + usage text).
   1.2 Builds the `new` command as an explicit Cobra command.
       1.2.1 Sets `Use` to `new <project-name> | new cli <project-name>`.
       1.2.2 Wires `Args: validateNewArgs` and `RunE: runNewCommand`.
       1.2.3 Assigns detailed `Long` help and usage examples for both legacy and preset syntax.
   1.3 Registers generation flags on `new`.
       1.3.1 Layout and module: `--flat`, `--module`.
       1.3.2 Integrations: `--sqlite`, `--postgres`, `--ai`.
       1.3.3 Config mode: `--local`, `--global`, `--config-scope`.
       1.3.4 Idempotency/safety: `--dry-run`, `--force`, `--skip-existing`.
       1.3.5 Post-generation check: `--verify`.
       1.3.6 Automation output: `--json`.
   1.4 Finalizes root command behavior.
       1.4.1 Adds `new` to the app.
       1.4.2 Hides Cobra completion command.
       1.4.3 Applies styled help templates to commands.
       1.4.4 Overrides root help template with minimal quick-start help (legacy + preset examples).
   1.5 Starts execution.
       1.5.1 Calls `app.Run()`.
       1.5.2 If non-nil error, exits process with `os.Exit(1)`.

2. Command execution layer (`cli/cli.go`)
   2.1 `App.Run()` calls Cobra `Execute()` on the root command.
   2.2 Cobra resolves argv to command path (`gokart new ...`).
   2.3 Root `PersistentPreRunE` runs `initConfig()`.
       2.3.1 If `--config` or `WithConfig` provided: `SetConfigFile(path)`.
       2.3.2 Else if config name configured: sets config name/type and search paths (`.`, `$HOME/.config/<app>`, `/etc/<app>`).
       2.3.3 Calls `ReadInConfig()`.
       2.3.4 Ignores only `ConfigFileNotFoundError`; returns any other config error.
   2.4 If pre-run succeeds, Cobra invokes the `new` command handler.

3. `gokart new` handler flow (`cmd/gokart/main.go`)
   3.1 Validates invocation shape.
       3.1.1 `validateNewArgs` calls `parseNewInvocation(args)`.
       3.1.2 Accepts both `gokart new <name>` and `gokart new cli <name>`.
       3.1.3 Produces targeted errors for ambiguous `gokart new cli`, unknown presets, or bad arg counts.
   3.2 Resolves runtime options.
       3.2.1 Reads flags: `flat`, `module`, `sqlite`, `postgres`, `ai`, `local`, `global`, `config-scope`, `dry-run`, `force`, `skip-existing`, `verify`, `json`.
       3.2.2 `resolveUseGlobal(...)` validates config-flag combinations and computes effective `useGlobal`.
       3.2.3 Emits warnings for no-op/legacy combinations (for example `--local` in flat mode).
       3.2.4 `resolveExistingFilePolicy(...)` maps overwrite behavior to `fail|skip|overwrite`.
   3.3 Normalizes and validates target inputs.
       3.3.1 `normalizeProjectArg(...)` trims/cleans input and derives `projectName` plus `targetDir`.
       3.3.2 Rejects invalid names (`.`, `..`, separator-only, empty, or regex mismatch).
       3.3.3 Defaults module path to `projectName` when `--module` is omitted.
       3.3.4 `validateModulePath(...)` validates slash segments and allowed characters.
       3.3.5 `validateTargetDir(...)` allows non-existent targets but rejects non-directory collisions.
   3.4 Executes scaffold plan by mode.
       3.4.1 Builds `ApplyOptions{DryRun, ExistingFilePolicy}`.
       3.4.2 Flat branch warns that integration flags are ignored, logs mode, and calls `ScaffoldFlat(..., opts)`.
       3.4.3 Structured branch logs mode and calls `ScaffoldStructured(..., opts)`.
   3.5 Reports results and optional verification.
       3.5.1 Prints dry-run or creation success message.
       3.5.2 `printApplyResult(...)` prints counts and per-file actions (`create/overwrite/skip/unchanged`).
       3.5.3 If `--dry-run --verify`, warns that verify is ignored.
       3.5.4 If `--verify` on real run, executes `go mod tidy` and `go test ./...` in target dir.
       3.5.5 Verify failures return explicit "project generated, but verification failed" errors.
       3.5.6 Prints quoted next step hint: `cd <targetDir> && go mod tidy`.
       3.5.7 If `--json`, emits machine-readable payloads for success/failure (with conflict lists and action results).

4. Scaffolding engine (`cmd/gokart/scaffold.go`, `cmd/gokart/scaffolder.go`)
   4.1 Entry functions build template context and dispatch.
       4.1.1 `ScaffoldFlat(..., opts)` sets `Name`, `Module`, `GoVersion`, `UseGlobal`.
       4.1.2 `ScaffoldStructured(..., opts)` sets same plus `UseSQLite`, `UsePostgres`, `UseAI`.
       4.1.3 Both call `Apply(templates, root, targetDir, data, opts)` and return `*ApplyResult`.
       4.1.4 `goVersion()` reads `runtime.Version()` and strips the `go` prefix.
   4.2 `Apply()` runs a plan/apply pipeline.
       4.2.1 `normalizeApplyOptions(...)` defaults unspecified policy to `fail`.
       4.2.2 `buildPlan(...)` renders templates and determines actions before any write.
       4.2.3 `collectResult(...)` gathers action summaries.
       4.2.4 Dry-run returns the plan result without filesystem mutation.
       4.2.5 Non-dry-run calls `applyPlanWrites(...)` for create/overwrite actions.
   4.3 `buildPlan()` per-file logic.
       4.3.1 Walks template tree via `fs.WalkDir` and skips directories.
       4.3.2 `templateOutputPath(...)` strips `.tmpl`, normalizes paths, and blocks path traversal.
       4.3.3 `renderTemplate(...)` reads/parses/executes templates with FuncMap (`upper`).
       4.3.4 Skips files whose rendered output is whitespace-only.
       4.3.5 Computes destination action:
           4.3.5.1 missing file -> `create`
           4.3.5.2 identical content -> `unchanged`
           4.3.5.3 different content -> `fail|skip|overwrite` by policy
           4.3.5.4 destination is directory -> error
       4.3.6 In `fail` mode, conflicting paths are aggregated into a single conflict error.
   4.4 Write path resiliency.
       4.4.1 `writePlannedFile(...)` ensures parent directories exist.
       4.4.2 `writeFileAtomic(...)` writes temp file, sets mode, then renames into place.
       4.4.3 Rename fallback handles platforms where direct replace is not allowed.
       4.4.4 Before each write, create/overwrite actions are revalidated against on-disk state.
       4.4.5 Rename fallback stages existing destination to backup and restores on failure.
       4.4.6 `applyPlanWrites(...)` records rollback actions for each successful mutation.
       4.4.7 On write failure, `rollbackWrites(...)` removes created files and restores overwritten content/mode.
       4.4.8 Rollback failures are joined and surfaced with the original error.

5. Exit outcomes
   5.1 Failure path.
       5.1.1 Any config/validation/scaffold/write/verify error is returned from handler.
       5.1.2 Cobra returns the error through `app.Run()`.
       5.1.3 `main()` exits with code 1.
   5.2 Success path.
       5.2.1 All templates are processed and written.
       5.2.2 Optional verify path (if enabled) passes `go mod tidy` and `go test ./...`.
       5.2.3 Handler returns nil.
       5.2.4 Process exits 0 after success output.
