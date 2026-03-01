1. Startup and command graph (`cmd/gokart/main.go`)
   1.1 Intent block: keep process entry minimal and deterministic.
       1.1.1 `main()` delegates to `run()`.
       1.1.2 `main()` maps returned typed errors through `exitCodeForError(...)` before `os.Exit(...)`.
   1.2 Intent block: compose the app in one place.
       1.2.1 `run()` builds the CLI via `newGokartApp(gokartVersion)` and calls `app.Run()`.
       1.2.2 `newGokartApp(...)` configures app identity/description and long usage text.
       1.2.3 `newGokartApp(...)` installs `newNewCommand()` and applies root-level wiring through `configureRootCommand(...)`.
   1.3 Intent block: define `new` command contract explicitly.
       1.3.1 `newNewCommand()` sets `Use`, `Short`, `Long`, and `Example` text.
       1.3.2 `newNewCommand()` wires argument validation via `validateNewArgs`.
       1.3.3 `newNewCommand()` wires execution via `runNewCommand`.
   1.4 Intent block: register flags from one source of truth.
       1.4.1 `configureNewCommandFlags(...)` defines layout/module flags: `--flat`, `--module`.
       1.4.2 `configureNewCommandFlags(...)` defines integrations: `--sqlite`, `--postgres`, `--ai`.
       1.4.3 `configureNewCommandFlags(...)` defines example scaffold opt-in: `--example`.
       1.4.4 `configureNewCommandFlags(...)` defines config scope controls: `--local`, `--global`, `--config-scope`.
       1.4.5 `configureNewCommandFlags(...)` defines idempotency/safety controls: `--dry-run`, `--force`, `--skip-existing`, `--no-manifest`.
       1.4.6 `configureNewCommandFlags(...)` defines verification controls: `--verify`, `--verify-only`, `--verify-timeout`.
       1.4.7 `configureNewCommandFlags(...)` defines automation output: `--json`.
   1.5 Intent block: finalize root command UX and error wrapping.
       1.5.1 `configureRootCommand(...)` hides Cobra completion from help.
       1.5.2 `configureRootCommand(...)` applies styled help templates to subcommands.
       1.5.3 `configureRootCommand(...)` sets a minimal root help template with quick-start examples.
       1.5.4 `configureRootCommand(...)` wraps `PersistentPreRunE` with JSON-aware config/init error handling.

2. Command execution layer (`cli/cli.go` + root wrapper)
   2.1 Intent block: execute through Cobra with predictable pre-run behavior.
       2.1.1 `App.Run()` calls root `Execute()`.
       2.1.2 Cobra resolves argv (`gokart new ...`) to command path and runs root pre-run first.
   2.2 Intent block: initialize config without hard-failing on missing files.
       2.2.1 `initConfig()` honors explicit config file when configured (`WithConfig`).
       2.2.2 `initConfig()` otherwise uses configured config name search paths (`.`, `$HOME/.config/<app>`, `/etc/<app>`) when set (`WithConfigName`).
       2.2.3 `initConfig()` calls `ReadInConfig()` and suppresses only `ConfigFileNotFoundError`.
       2.2.4 Any other config parse/read failure is returned upward.
   2.3 Intent block: keep styled help setup idempotent and safe.
       2.3.1 `SetStyledHelp(...)` is nil-safe.
       2.3.2 Template helper function registration is guarded by `sync.Once` to avoid duplicate global registration.
   2.4 Intent block: normalize pre-run failures for humans and automation.
       2.4.1 `wrapPersistentPreRunJSONErrors(...)` intercepts root pre-run errors.
       2.4.2 `handlePersistentPreRunError(...)` wraps non-typed pre-run errors as `config_init_failed` (`exit_code=6`).
       2.4.3 When `--json` is enabled, pre-run failures suppress Cobra noise and emit one JSON payload.

3. `gokart new` request lifecycle (`cmd/gokart/main.go`)
   3.1 Intent block: validate invocation shape early.
       3.1.1 `parseNewInvocation(...)` accepts `gokart new <name>` and `gokart new cli <name>`.
       3.1.2 It rejects ambiguous `gokart new cli`, unknown presets, and invalid arg counts with targeted errors.
       3.1.3 `validateNewArgs(...)` converts failures into typed `invalid_arguments` (`exit_code=2`) and emits JSON when requested.
   3.2 Intent block: materialize a normalized request object.
       3.2.1 `buildNewRequest(...)` reads all flags into `newRequest` (including `IncludeExample` for `--example`).
       3.2.2 It rejects negative `--verify-timeout`.
       3.2.3 It derives mode (`flat` or `structured`) and defaults `module` to project name.
       3.2.4 It sets `WriteManifest` as inverse of `--no-manifest`.
   3.3 Intent block: enforce mode-specific argument/flag semantics.
       3.3.1 Verify-only path forbids `--dry-run`, forces `Verify=true`, and forces existing-file policy to `fail`.
       3.3.2 Verify-only path computes effective `UseGlobal` with auto scope semantics and records ignored generation flags.
       3.3.3 Verify-only path requires an existing target directory (`requireExistingTargetDir`).
       3.3.4 Normal path resolves global/local config scope (`resolveUseGlobal`) with warning support for no-op legacy combos.
       3.3.5 Normal path resolves overwrite policy (`resolveExistingFilePolicy`) and validates module path/target directory.
   3.4 Intent block: execute either verify-only or scaffolding.
       3.4.1 Verify-only branch skips templating/writes and runs verification directly in target dir.
       3.4.2 Scaffold branch builds `ApplyOptions{DryRun, ExistingFilePolicy, SkipManifest}` and forwards `IncludeExample` to scaffolders.
       3.4.3 Flat scaffold branch warns when integration flags are supplied and ignored.
       3.4.4 Structured scaffold branch forwards integration flags to `ScaffoldStructured(...)`.
       3.4.5 Scaffold errors are classified into conflict (`existing_file_conflict`), lock contention (`target_locked`), or generic scaffold failure.
   3.5 Intent block: run verification consistently across modes.
       3.5.1 `runVerifyForRequest(...)` routes verify-only and real-run verification to `runVerify(...)`.
       3.5.2 `runVerifyForRequest(...)` routes dry-run verification through `runDryRunVerify(...)`.
       3.5.3 `runDryRunVerify(...)` scaffolds into a temporary directory with writes enabled, verifies there, then cleans up.
       3.5.4 `runVerify(...)` executes `go mod tidy` then `go test ./...` under optional timeout context.
       3.5.5 `runCommand(...)` captures output in non-verbose mode and truncates long failure output.
   3.6 Intent block: emit stable human + machine output.
       3.6.1 Success JSON is seeded from request data (`newCommandOutputFromRequest(...)`).
       3.6.2 Non-JSON mode prints status/warnings and per-file action summaries (`printApplyResult`).
       3.6.3 Non-dry-run success adds both human `next_command` and structured `next` metadata.
       3.6.4 `failNewCommand(...)` sets `outcome`, `error_code`, `exit_code`, `error`, and conflict paths for automation.

4. Scaffolding transaction engine (`cmd/gokart/scaffold.go`, `cmd/gokart/scaffolder.go`)
   4.1 Intent block: build template context and dispatch templates.
       4.1.1 `ScaffoldFlat(...)` builds baseline template data and applies `templates/flat`.
       4.1.2 `ScaffoldStructured(...)` extends baseline data with integration booleans and applies `templates/structured`.
       4.1.3 `baseTemplateData(...)` injects Go version, pinned dependency versions, and `IncludeExample` template toggle.
   4.2 Intent block: normalize options and orchestrate plan/apply phases.
       4.2.1 `normalizeApplyOptions(...)` defaults policy to `fail` and validates policy value.
       4.2.2 `Apply(...)` resolves absolute target root before any mutation.
       4.2.3 Non-dry-run `Apply(...)` acquires a lock, recovers pending journals, builds plan, writes with journal, marks complete, and cleans journal.
       4.2.4 Dry-run `Apply(...)` returns planned actions without writing.
   4.3 Intent block: create a safe deterministic write plan.
       4.3.1 `buildPlan(...)` walks templates, renders each file, skips whitespace-only render output.
       4.3.2 `templateOutputPath(...)` strips `.tmpl` and blocks traversal-like outputs.
       4.3.3 `safeDestinationPath(...)` and `ensureNoSymlinkFromRoot(...)` enforce root confinement and symlink resistance.
       4.3.4 Existing destination classification yields `create`, `unchanged`, `skip`, or `overwrite`.
       4.3.5 Policy `fail` aggregates conflicts into one `ExistingFileConflictError`.
   4.4 Intent block: apply writes with rollback intent recorded first.
       4.4.1 `applyPlanWrites(...)` appends rollback actions to journal before each mutation.
       4.4.2 Create path records rollback-remove expectation and writes file atomically.
       4.4.3 Overwrite path records rollback-restore content/mode and writes atomically.
       4.4.4 Each applied action is marked in journal only after mutation succeeds.
       4.4.5 Real-run result `Created`/`Overwritten` is populated during this phase.
   4.5 Intent block: emit scaffold manifest as part of transaction.
       4.5.1 Unless `SkipManifest`, manifest content is rendered (`.gokart-manifest.json`) from full plan.
       4.5.2 Manifest entries include action, template/content SHA256 hashes, and mode.
       4.5.3 Manifest write has rollback action and journal tracking like any other mutation.
   4.6 Intent block: guarantee recoverability across crashes/failures.
       4.6.1 `beginApplyJournal(...)` creates `.gokart/tx/*.json` journal files.
       4.6.2 `rollbackWithError(...)` rolls back in-memory applied actions and finalizes journal state.
       4.6.3 `recoverPendingJournals(...)` replays unfinished rollback actions on next run.
       4.6.4 Recovery validates expected generated-file state (`verifyJournalActionExpectedState(...)`) before destructive rollback and blocks on mismatch.
       4.6.5 Journal v1 payloads are upgraded on load to current semantics.
   4.7 Intent block: serialize concurrent writers and reclaim stale locks.
       4.7.1 `acquireApplyLock(...)` creates `.gokart.lock` atomically and returns a release closure.
       4.7.2 Lock metadata stores PID/timestamp/stale threshold for reclaim decisions.
       4.7.3 `shouldReclaimStaleLock(...)` reclaims when PID is dead or lock age exceeds policy.
       4.7.4 Active, non-stale lock contention returns typed `ApplyLockError`.
   4.8 Intent block: make each file write durable and reversible.
       4.8.1 `writeFileAtomic(...)` writes temp file, chmods, fsyncs, then renames.
       4.8.2 Rename fallback stages prior destination to backup and restores on failure.
       4.8.3 Directory sync is attempted best-effort (`syncDirBestEffort(...)`) with platform-tolerant error handling.

5. Outcomes, exit codes, and JSON contract
   5.1 Intent block: define success semantics clearly.
       5.1.1 `outcome=success`, `exit_code=0` covers dry-run success, scaffold success, and verify-only success.
       5.1.2 Success JSON carries request echo fields, verify fields, file action result, and optional `next` metadata.
   5.2 Intent block: preserve partial-success semantics for generated-but-not-verified projects.
       5.2.1 Post-generation verify failure returns `outcome=partial_success`, `error_code=verify_failed`, `exit_code=4`.
       5.2.2 The scaffold `result` remains populated for automation inspection.
   5.3 Intent block: map failures to stable typed outcomes.
       5.3.1 Invalid invocation/flag/input validation -> `invalid_arguments` (`exit_code=2`).
       5.3.2 Existing-file conflicts -> `existing_file_conflict` (`exit_code=3`).
       5.3.3 Verify-only or dry-run verification failure -> `verify_failed` (`exit_code=4`).
       5.3.4 Target lock contention -> `target_locked` (`exit_code=5`).
       5.3.5 Root config/init pre-run failure -> `config_init_failed` (`exit_code=6`).
       5.3.6 Other scaffold/apply/write/rollback failures -> `scaffold_failed` (`exit_code=7`).
       5.3.7 JSON encode failure in command-output paths -> `json_encode_failed` (`exit_code=8`).
       5.3.8 Untyped fallthrough errors map to generic process `exit_code=1`.
   5.4 Intent block: make JSON mode single-payload and automation-safe.
       5.4.1 `--json` suppresses Cobra usage/error noise for command and pre-run wrappers.
       5.4.2 `emitJSON(...)` emits one indented JSON object to stdout.
       5.4.3 JSON encoding failures in `failNewCommand(...)` and success emission are surfaced as typed command errors.
       5.4.4 JSON emission failures in early validation/pre-run wrappers are logged to stderr while original typed exit codes are preserved.
