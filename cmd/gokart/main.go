package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

const (
	defaultPreset = "cli"

	configScopeAuto   = "auto"
	configScopeLocal  = "local"
	configScopeGlobal = "global"
)

var (
	projectNamePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	moduleSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

const logo = `
   ____       _  __          _
  / ___| ___ | |/ /__ _ _ __| |_
 | |  _ / _ \| ' // _' | '__| __|
 | |_| | (_) | . \ (_| | |  | |_
  \____|\___/|_|\_\__,_|_|   \__|
`

func main() {
	app := cli.NewApp("gokart", "0.1.0").
		WithDescription("Scaffold Go service projects").
		WithLongDescription(logo + `
gokart new <name> [flags]
gokart new cli <name> [flags]

  --sqlite         SQLite database (modernc.org/sqlite)
  --postgres       PostgreSQL pool (pgx/v5)
  --ai             OpenAI client (openai-go/v3)
  --flat           Single main.go (no internal/)
  --local          No global config (structured: default is global)
  --global         Global config (flat: default is local)
  --config-scope   Config scope: auto|local|global
  --module         Custom module path
  --dry-run        Preview scaffold operations without writing files
  --force          Overwrite existing generated files
  --skip-existing  Keep existing files and write only missing ones
  --verify         Run go mod tidy and go test ./... after generation
  --json           Print machine-readable JSON result`)

	newCmd := &cobra.Command{
		Use:   "new <project-name> | new cli <project-name>",
		Short: "Create a new GoKart project",
		Args:  cobra.ArbitraryArgs,
		RunE:  runNewCommand,
	}

	newCmd.Long = `Create a new Go project with sensible defaults and optional integrations.

Structured mode (default) creates:
  myapp/
  ├── main.go                    # Entry point
  ├── internal/
  │   ├── commands/              # Cobra command definitions
  │   └── actions/               # Business logic
  └── go.mod

Flat mode creates a single main.go for quick scripts.`

	newCmd.Example = `  # Basic structured project
  gokart new myapi

  # Explicit preset (same output as command above)
  gokart new cli myapi

  # With PostgreSQL and OpenAI
  gokart new myapi --postgres --ai

  # With SQLite for local-first CLI
  gokart new mycli --sqlite

  # Quick script (single main.go)
  gokart new script --flat

  # Preview without writing files
  gokart new myapi --dry-run

  # Overwrite existing generated files
  gokart new myapi --force

  # Generate and verify immediately
  gokart new myapi --verify

  # JSON output for CI tooling
  gokart new myapi --dry-run --json

  # Custom module path
  gokart new myapi --module github.com/myorg/myapi`

	newCmd.Flags().Bool("flat", false, "Use flat structure (single main.go)")
	newCmd.Flags().String("module", "", "Go module path (defaults to project name)")
	newCmd.Flags().Bool("sqlite", false, "Include SQLite database wiring (modernc.org/sqlite)")
	newCmd.Flags().Bool("postgres", false, "Include PostgreSQL connection pool (pgx/v5)")
	newCmd.Flags().Bool("ai", false, "Include OpenAI client (openai-go/v3)")
	newCmd.Flags().Bool("local", false, "Disable global config (structured only, default is global)")
	newCmd.Flags().Bool("global", false, "Enable global config (flat only, default is local)")
	newCmd.Flags().String("config-scope", configScopeAuto, "Config scope: auto|local|global")
	newCmd.Flags().Bool("dry-run", false, "Preview scaffold operations without writing files")
	newCmd.Flags().Bool("force", false, "Overwrite existing generated files")
	newCmd.Flags().Bool("skip-existing", false, "Keep existing files and only generate missing files")
	newCmd.Flags().Bool("verify", false, "Run go mod tidy and go test ./... after generation")
	newCmd.Flags().Bool("json", false, "Print machine-readable JSON result")

	app.AddCommand(newCmd)

	// Hide completion command from help
	for _, cmd := range app.Root().Commands() {
		if cmd.Name() == "completion" {
			cmd.Hidden = true
		}
		cli.SetStyledHelp(cmd)
	}

	// Minimal root help - just show Long description
	app.Root().SetHelpTemplate(`{{.Long}}

  gokart new myapp
  gokart new cli myapp
  gokart new myapp --postgres --ai

  gokart new --help    Full options
`)

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}

type newCommandOutput struct {
	Preset             string             `json:"preset,omitempty"`
	Mode               string             `json:"mode,omitempty"`
	ProjectName        string             `json:"project_name,omitempty"`
	TargetDir          string             `json:"target_dir,omitempty"`
	Module             string             `json:"module,omitempty"`
	ConfigScope        string             `json:"config_scope,omitempty"`
	UseGlobal          bool               `json:"use_global"`
	DryRun             bool               `json:"dry_run"`
	VerifyRequested    bool               `json:"verify_requested"`
	VerifyRan          bool               `json:"verify_ran"`
	VerifyPassed       bool               `json:"verify_passed"`
	ExistingFilePolicy ExistingFilePolicy `json:"existing_file_policy,omitempty"`
	Warnings           []string           `json:"warnings,omitempty"`
	Conflicts          []string           `json:"conflicts,omitempty"`
	Result             *ApplyResult       `json:"result,omitempty"`
	NextCommand        string             `json:"next_command,omitempty"`
	Error              string             `json:"error,omitempty"`
}

func runNewCommand(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	output := newCommandOutput{}

	preset, projectArg, err := parseNewInvocation(args)
	if err != nil {
		return failNewCommand(err, jsonOutput, &output)
	}
	output.Preset = preset

	if preset != defaultPreset {
		return failNewCommand(fmt.Errorf("unsupported preset %q", preset), jsonOutput, &output)
	}

	flat, _ := cmd.Flags().GetBool("flat")
	module, _ := cmd.Flags().GetString("module")
	sqlite, _ := cmd.Flags().GetBool("sqlite")
	postgres, _ := cmd.Flags().GetBool("postgres")
	ai, _ := cmd.Flags().GetBool("ai")
	local, _ := cmd.Flags().GetBool("local")
	global, _ := cmd.Flags().GetBool("global")
	configScope, _ := cmd.Flags().GetString("config-scope")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	skipExisting, _ := cmd.Flags().GetBool("skip-existing")
	verify, _ := cmd.Flags().GetBool("verify")

	output.DryRun = dryRun
	output.VerifyRequested = verify
	output.ConfigScope = configScope

	useGlobal, warnings, err := resolveUseGlobal(flat, local, global, configScope)
	if err != nil {
		return failNewCommand(err, jsonOutput, &output)
	}
	output.UseGlobal = useGlobal
	output.Warnings = append(output.Warnings, warnings...)
	for _, warning := range warnings {
		if !jsonOutput {
			cli.Warning("%s", warning)
		}
	}

	existingPolicy, err := resolveExistingFilePolicy(force, skipExisting)
	if err != nil {
		return failNewCommand(err, jsonOutput, &output)
	}
	output.ExistingFilePolicy = existingPolicy

	projectName, targetDir, err := normalizeProjectArg(projectArg)
	if err != nil {
		return failNewCommand(err, jsonOutput, &output)
	}
	output.ProjectName = projectName
	output.TargetDir = targetDir

	if module == "" {
		module = projectName
	}
	output.Module = module
	if err := validateModulePath(module); err != nil {
		return failNewCommand(fmt.Errorf("invalid module path %q: %w", module, err), jsonOutput, &output)
	}

	if err := validateTargetDir(targetDir); err != nil {
		return failNewCommand(err, jsonOutput, &output)
	}

	opts := ApplyOptions{
		DryRun:             dryRun,
		ExistingFilePolicy: existingPolicy,
	}

	var result *ApplyResult
	mode := "structured"
	if flat {
		mode = "flat"
		if sqlite || postgres || ai {
			flatWarning := "--sqlite, --postgres, and --ai flags are ignored in flat mode"
			output.Warnings = append(output.Warnings, flatWarning)
			if !jsonOutput {
				cli.Warning("%s", flatWarning)
			}
		}
		if dryRun && !jsonOutput {
			cli.Info("Dry run: planning flat project (%s preset): %s", preset, projectName)
		} else if !jsonOutput {
			cli.Info("Scaffolding flat project (%s preset): %s", preset, projectName)
		}
		result, err = ScaffoldFlat(targetDir, projectName, module, useGlobal, opts)
	} else {
		if dryRun && !jsonOutput {
			cli.Info("Dry run: planning structured project (%s preset): %s", preset, projectName)
		} else if !jsonOutput {
			cli.Info("Scaffolding structured project (%s preset): %s", preset, projectName)
		}
		result, err = ScaffoldStructured(targetDir, projectName, module, sqlite, postgres, ai, useGlobal, opts)
	}
	output.Mode = mode
	if err != nil {
		var conflictErr *ExistingFileConflictError
		if errors.As(err, &conflictErr) {
			output.Conflicts = append([]string(nil), conflictErr.Paths...)
			if !jsonOutput {
				cli.Warning("Found %d conflicting existing file(s):", len(conflictErr.Paths))
				for _, path := range conflictErr.Paths {
					cli.Dim("  conflict   %s", path)
				}
			}
		}
		return failNewCommand(err, jsonOutput, &output)
	}
	output.Result = result

	if dryRun {
		if !jsonOutput {
			cli.Success("Dry run complete for %s", targetDir)
		}
	} else {
		if !jsonOutput {
			cli.Success("Project created at %s", targetDir)
		}
	}

	if !jsonOutput {
		printApplyResult(result, dryRun)
	}

	if dryRun && verify {
		warning := "--verify is ignored when --dry-run is set"
		output.Warnings = append(output.Warnings, warning)
		if !jsonOutput {
			cli.Warning("%s", warning)
		}
	}

	if !dryRun && verify {
		output.VerifyRan = true
		if err := runVerify(targetDir, !jsonOutput); err != nil {
			output.VerifyPassed = false
			return failNewCommand(fmt.Errorf("project generated at %s, but verification failed: %w", targetDir, err), jsonOutput, &output)
		}
		output.VerifyPassed = true
		if !jsonOutput {
			cli.Success("Verification passed")
		}
	}

	if !dryRun {
		next := fmt.Sprintf("cd %s && go mod tidy", shellQuote(targetDir))
		output.NextCommand = next
		if !jsonOutput {
			cli.Dim("  %s", next)
		}
	}

	if jsonOutput {
		if err := emitJSON(output); err != nil {
			return fmt.Errorf("encode JSON output: %w", err)
		}
	}

	return nil
}

func failNewCommand(err error, jsonOutput bool, output *newCommandOutput) error {
	if jsonOutput {
		if output != nil {
			output.Error = err.Error()

			var conflictErr *ExistingFileConflictError
			if errors.As(err, &conflictErr) && len(output.Conflicts) == 0 {
				output.Conflicts = append([]string(nil), conflictErr.Paths...)
			}

			if emitErr := emitJSON(output); emitErr != nil {
				fmt.Fprintf(os.Stderr, "failed to write JSON output: %v\n", emitErr)
			}
		}
	}

	return err
}

func emitJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

func parseNewInvocation(args []string) (preset string, projectArg string, err error) {
	switch len(args) {
	case 1:
		if strings.EqualFold(strings.TrimSpace(args[0]), defaultPreset) {
			return "", "", fmt.Errorf("missing project name: use `gokart new %s <project-name>` (or `gokart new ./%s` to create a project named %s)", defaultPreset, defaultPreset, defaultPreset)
		}
		return defaultPreset, args[0], nil
	case 2:
		preset := strings.ToLower(strings.TrimSpace(args[0]))
		if preset != defaultPreset {
			return "", "", fmt.Errorf("unknown preset %q (supported presets: %s)", args[0], defaultPreset)
		}
		return preset, args[1], nil
	default:
		return "", "", fmt.Errorf("usage: gokart new <project-name> or gokart new %s <project-name>", defaultPreset)
	}
}

func resolveUseGlobal(flat, local, global bool, configScope string) (bool, []string, error) {
	if local && global {
		return false, nil, fmt.Errorf("cannot use --local and --global together")
	}

	scope := strings.ToLower(strings.TrimSpace(configScope))
	if scope == "" {
		scope = configScopeAuto
	}

	if scope != configScopeAuto && (local || global) {
		return false, nil, fmt.Errorf("cannot combine --config-scope with --local or --global")
	}

	switch scope {
	case configScopeAuto:
		warnings := make([]string, 0, 1)
		if flat {
			if local {
				warnings = append(warnings, "--local has no effect in flat mode")
			}
			return global, warnings, nil
		}

		if global {
			warnings = append(warnings, "--global is already the default in structured mode")
		}
		return !local, warnings, nil
	case configScopeLocal:
		return false, nil, nil
	case configScopeGlobal:
		return true, nil, nil
	default:
		return false, nil, fmt.Errorf("invalid --config-scope %q (valid values: auto, local, global)", configScope)
	}
}

func resolveExistingFilePolicy(force, skipExisting bool) (ExistingFilePolicy, error) {
	if force && skipExisting {
		return "", fmt.Errorf("cannot use --force and --skip-existing together")
	}

	if force {
		return ExistingFilePolicyOverwrite, nil
	}

	if skipExisting {
		return ExistingFilePolicySkip, nil
	}

	return ExistingFilePolicyFail, nil
}

func normalizeProjectArg(projectArg string) (projectName, targetDir string, err error) {
	raw := strings.TrimSpace(projectArg)
	if raw == "" {
		return "", "", fmt.Errorf("project name is required")
	}

	cleanArg := filepath.Clean(raw)
	projectName = filepath.Base(cleanArg)

	if projectName == "." || projectName == ".." || projectName == string(filepath.Separator) || projectName == "" {
		return "", "", fmt.Errorf("invalid project name %q", projectArg)
	}

	if !projectNamePattern.MatchString(projectName) {
		return "", "", fmt.Errorf("invalid project name %q (allowed: letters, numbers, ., _, -)", projectName)
	}

	if filepath.IsAbs(raw) {
		targetDir = cleanArg
	} else {
		targetDir = filepath.Join(".", cleanArg)
	}

	return projectName, targetDir, nil
}

func validateModulePath(module string) error {
	mod := strings.TrimSpace(module)
	if mod == "" {
		return fmt.Errorf("cannot be empty")
	}

	if strings.ContainsAny(mod, " \t\r\n") {
		return fmt.Errorf("cannot contain whitespace")
	}

	if strings.HasPrefix(mod, "/") || strings.HasSuffix(mod, "/") {
		return fmt.Errorf("cannot start or end with '/'")
	}

	for _, segment := range strings.Split(mod, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("contains invalid path segment %q", segment)
		}
		if !moduleSegmentPattern.MatchString(segment) {
			return fmt.Errorf("contains invalid path segment %q", segment)
		}
	}

	return nil
}

func validateTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path %q exists and is not a directory", targetDir)
	}

	return nil
}

func printApplyResult(result *ApplyResult, dryRun bool) {
	if result == nil {
		return
	}

	label := "Applied"
	if dryRun {
		label = "Planned"
	}

	cli.Info("%s: %d create, %d overwrite, %d skip, %d unchanged", label, len(result.Created), len(result.Overwritten), len(result.Skipped), len(result.Unchanged))
	printResultPaths("create", result.Created)
	printResultPaths("overwrite", result.Overwritten)
	printResultPaths("skip", result.Skipped)
	printResultPaths("unchanged", result.Unchanged)
}

func printResultPaths(label string, paths []string) {
	for _, path := range paths {
		cli.Dim("  %-10s %s", label, path)
	}
}

func shellQuote(path string) string {
	if strings.ContainsAny(path, " \t\n\"'`$()[]{}&|;<>*?!") {
		return strconv.Quote(path)
	}
	return path
}

func runVerify(targetDir string, verbose bool) error {
	if verbose {
		cli.Info("Verifying generated project in %s", targetDir)
	}

	if err := runCommand(targetDir, verbose, "go", "mod", "tidy"); err != nil {
		return err
	}

	if err := runCommand(targetDir, verbose, "go", "test", "./..."); err != nil {
		return err
	}

	return nil
}

func runCommand(dir string, verbose bool, name string, args ...string) error {
	if verbose {
		cli.Info("Running: %s", strings.Join(append([]string{name}, args...), " "))
	}

	execCmd := exec.Command(name, args...)
	execCmd.Dir = dir
	if verbose {
		execCmd.Stdout = os.Stdout
	} else {
		execCmd.Stdout = os.Stderr
	}
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", strings.Join(append([]string{name}, args...), " "), err)
	}

	return nil
}
