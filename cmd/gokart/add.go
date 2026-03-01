package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

const (
	addFlagDryRun        = "dry-run"
	addFlagForce         = "force"
	addFlagJSON          = "json"
	addFlagVerify        = "verify"
	addFlagVerifyTimeout = "verify-timeout"

	errorCodeManifestNotFound    commandErrorCode = "manifest_not_found"
	errorCodeFlatModeUnsupported commandErrorCode = "flat_mode_unsupported"

	exitCodeManifestNotFound = 9
	exitCodeFlatUnsupported  = 10

	integrationSQLite   = "sqlite"
	integrationPostgres = "postgres"
	integrationAI       = "ai"
)

var validIntegrations = map[string]bool{
	integrationSQLite:   true,
	integrationPostgres: true,
	integrationAI:       true,
}

type integrationDep struct {
	Packages []string
}

var integrationDeps = map[string]integrationDep{
	integrationSQLite:   {Packages: []string{"github.com/dotcommander/gokart/sqlite@latest"}},
	integrationPostgres: {Packages: []string{"github.com/dotcommander/gokart/postgres@latest", "github.com/jackc/pgx/v5@latest"}},
	integrationAI:       {Packages: []string{"github.com/dotcommander/gokart/ai@latest", "github.com/openai/openai-go/v3@latest"}},
}

type addCommandOutput struct {
	Outcome          commandOutcome   `json:"outcome,omitempty"`
	ErrorCode        commandErrorCode `json:"error_code,omitempty"`
	ExitCode         int              `json:"exit_code"`
	Integrations     []string         `json:"integrations,omitempty"`
	Added            []string         `json:"added,omitempty"`
	AlreadyPresent   []string         `json:"already_present,omitempty"`
	FilesCreated     []string         `json:"files_created,omitempty"`
	FilesOverwritten []string         `json:"files_overwritten,omitempty"`
	DryRun           bool             `json:"dry_run"`
	VerifyRequested  bool             `json:"verify_requested"`
	VerifyPassed     bool             `json:"verify_passed"`
	Warnings         []string         `json:"warnings,omitempty"`
	Error            string           `json:"error,omitempty"`
}

type fileSafetyResult int

const (
	fileSafetyCreate   fileSafetyResult = iota // file doesn't exist → create
	fileSafetySafe                             // hash matches manifest → safe overwrite
	fileSafetyConflict                         // hash differs → conflict
)

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <integration>...",
		Short: "Add integrations to an existing GoKart project",
		Long:  "Add SQLite, PostgreSQL, or OpenAI integrations to an existing structured project.\nRe-renders only integration-affected files (context.go, root.go) and runs go get.",
		Example: `  gokart add sqlite
  gokart add ai
  gokart add sqlite ai
  gokart add postgres --dry-run
  gokart add ai --force
  gokart add ai --json
  gokart add ai --verify`,
		Args: cobra.MinimumNArgs(1),
		RunE: runAddCommand,
	}

	flags := cmd.Flags()
	flags.Bool(addFlagDryRun, false, "Preview changes without writing files")
	flags.Bool(addFlagForce, false, "Overwrite user-modified files")
	flags.Bool(addFlagJSON, false, "Print machine-readable JSON result")
	flags.Bool(addFlagVerify, false, "Run go test ./... after adding")
	flags.Duration(addFlagVerifyTimeout, defaultVerifyTimeout, "Maximum time for --verify commands")

	return cmd
}

func runAddCommand(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool(addFlagJSON)
	if jsonOutput {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
	}

	dryRun, _ := cmd.Flags().GetBool(addFlagDryRun)
	force, _ := cmd.Flags().GetBool(addFlagForce)
	verify, _ := cmd.Flags().GetBool(addFlagVerify)
	verifyTimeout, _ := cmd.Flags().GetDuration(addFlagVerifyTimeout)

	output := addCommandOutput{
		Outcome: commandOutcomeSuccess,
		DryRun:  dryRun,
	}

	dir, err := os.Getwd()
	if err != nil {
		return failAddCommand(fmt.Errorf("get working directory: %w", err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	// Validate and deduplicate integration names
	seen := make(map[string]bool, len(args))
	var requested []string
	for _, arg := range args {
		name := strings.ToLower(strings.TrimSpace(arg))
		if !validIntegrations[name] {
			return failAddCommand(fmt.Errorf("unknown integration: %s (valid: sqlite, postgres, ai)", name), jsonOutput, &output, errorCodeInvalidArguments, exitCodeInvalidArguments)
		}
		if !seen[name] {
			seen[name] = true
			requested = append(requested, name)
		}
	}
	output.Integrations = requested

	// Read manifest
	manifest, err := readAddManifest(dir)
	if err != nil {
		return failAddCommand(err, jsonOutput, &output, errorCodeManifestNotFound, exitCodeManifestNotFound)
	}

	// Reject flat projects
	if isFlatProject(manifest) {
		return failAddCommand(fmt.Errorf("gokart add requires a structured project (flat projects don't support integrations)"), jsonOutput, &output, errorCodeFlatModeUnsupported, exitCodeFlatUnsupported)
	}

	// Detect current integrations
	goModModule, current := detectCurrentIntegrations(manifest, dir)

	// Check which are new vs already present
	var toAdd []string
	for _, name := range requested {
		if integrationAlreadyEnabled(current, name) {
			output.AlreadyPresent = append(output.AlreadyPresent, name)
		} else {
			toAdd = append(toAdd, name)
		}
	}

	if len(toAdd) == 0 {
		for _, name := range output.AlreadyPresent {
			if !jsonOutput {
				cli.Warning("%s already enabled", name)
			}
		}
		if jsonOutput {
			_ = emitJSON(output)
		}
		return nil
	}
	output.Added = toAdd

	// Reconstruct TemplateData with merged integrations
	data, err := inferTemplateData(manifest, dir, current, toAdd, goModModule)
	if err != nil {
		return failAddCommand(fmt.Errorf("infer project state: %w", err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	// Render integration-affected templates
	renderedFiles, err := renderIntegrationFiles(data)
	if err != nil {
		return failAddCommand(fmt.Errorf("render templates: %w", err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	// Sort rendered file paths for deterministic output
	renderedPaths := make([]string, 0, len(renderedFiles))
	for relPath := range renderedFiles {
		renderedPaths = append(renderedPaths, relPath)
	}
	sort.Strings(renderedPaths)

	// Safety check each file
	for _, relPath := range renderedPaths {
		safety := checkFileSafety(dir, relPath, manifest)
		switch safety {
		case fileSafetyCreate:
			output.FilesCreated = append(output.FilesCreated, relPath)
		case fileSafetySafe:
			output.FilesOverwritten = append(output.FilesOverwritten, relPath)
		case fileSafetyConflict:
			if !force {
				return failAddCommand(
					fmt.Errorf("file %s has been modified (use --force to overwrite)", relPath),
					jsonOutput, &output, errorCodeExistingFileConflict, exitCodeExistingFileConflict,
				)
			}
			output.FilesOverwritten = append(output.FilesOverwritten, relPath)
			output.Warnings = append(output.Warnings, fmt.Sprintf("force-overwriting modified file: %s", relPath))
		}
	}

	// Dry-run exits here
	if dryRun {
		if !jsonOutput {
			cli.Info("Dry run: would add %s", strings.Join(toAdd, ", "))
			for _, p := range output.FilesCreated {
				cli.Dim("  create     %s", p)
			}
			for _, p := range output.FilesOverwritten {
				cli.Dim("  overwrite  %s", p)
			}
		}
		if jsonOutput {
			_ = emitJSON(output)
		}
		return nil
	}

	// Write files
	for _, relPath := range renderedPaths {
		destPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return failAddCommand(fmt.Errorf("create directory for %s: %w", relPath, err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}
		if err := writeFileAtomic(destPath, renderedFiles[relPath], 0644); err != nil {
			return failAddCommand(fmt.Errorf("write %s: %w", relPath, err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
		}
	}

	// Run go get for new dependencies
	if err := addGoDependencies(dir, toAdd, !jsonOutput); err != nil {
		return failAddCommand(fmt.Errorf("add dependencies: %w", err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	// Update manifest
	if err := updateAddManifest(dir, manifest, data, renderedFiles); err != nil {
		return failAddCommand(fmt.Errorf("update manifest: %w", err), jsonOutput, &output, errorCodeScaffoldFailed, exitCodeScaffoldFailed)
	}

	if !jsonOutput {
		cli.Success("Added %s", strings.Join(toAdd, ", "))
		for _, p := range output.FilesCreated {
			cli.Dim("  create     %s", p)
		}
		for _, p := range output.FilesOverwritten {
			cli.Dim("  overwrite  %s", p)
		}
		for _, w := range output.Warnings {
			cli.Warning("%s", w)
		}
	}

	// Verify if requested
	if verify {
		output.VerifyRequested = true
		if !jsonOutput {
			cli.Info("Verifying project...")
		}
		if err := runVerify(dir, verifyTimeout, !jsonOutput); err != nil {
			output.VerifyPassed = false
			return failAddCommand(fmt.Errorf("verification failed: %w", err), jsonOutput, &output, errorCodeVerifyFailed, exitCodeVerifyFailed)
		}
		output.VerifyPassed = true
		if !jsonOutput {
			cli.Success("Verification passed")
		}
	}

	if jsonOutput {
		_ = emitJSON(output)
	}

	return nil
}

func isFlatProject(manifest *scaffoldManifest) bool {
	return manifest.TemplateRoot == "templates/flat" || manifest.Mode == "flat"
}

func readAddManifest(dir string) (*scaffoldManifest, error) {
	path := filepath.Join(dir, scaffoldManifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no manifest found at %s (run gokart new to create a project first)", path)
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest scaffoldManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

func detectCurrentIntegrations(manifest *scaffoldManifest, dir string) (string, *manifestIntegrations) {
	// v2 manifests have integrations directly
	if manifest.Integrations != nil {
		return manifest.Module, manifest.Integrations
	}

	// v1: infer from go.mod
	return inferIntegrationsFromGoMod(dir)
}

func inferIntegrationsFromGoMod(dir string) (string, *manifestIntegrations) {
	result := &manifestIntegrations{}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", result
	}
	content := string(data)

	var module string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			module = strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			break
		}
	}

	if strings.Contains(content, "gokart/sqlite") {
		result.SQLite = true
	}
	if strings.Contains(content, "gokart/postgres") {
		result.Postgres = true
	}
	if strings.Contains(content, "gokart/ai") {
		result.AI = true
	}
	return module, result
}

func integrationAlreadyEnabled(current *manifestIntegrations, name string) bool {
	if current == nil {
		return false
	}
	switch name {
	case integrationSQLite:
		return current.SQLite
	case integrationPostgres:
		return current.Postgres
	case integrationAI:
		return current.AI
	default:
		return false
	}
}

func mergeIntegrations(current *manifestIntegrations, toAdd []string) *manifestIntegrations {
	merged := &manifestIntegrations{}
	if current != nil {
		*merged = *current
	}
	for _, name := range toAdd {
		switch name {
		case integrationSQLite:
			merged.SQLite = true
		case integrationPostgres:
			merged.Postgres = true
		case integrationAI:
			merged.AI = true
		}
	}
	return merged
}

func inferTemplateData(manifest *scaffoldManifest, dir string, current *manifestIntegrations, toAdd []string, goModModule string) (TemplateData, error) {
	// Get module from manifest or go.mod
	module := manifest.Module
	if module == "" {
		module = goModModule
		if module == "" {
			return TemplateData{}, fmt.Errorf("no module found in manifest or go.mod")
		}
	}

	name := filepath.Base(module)

	// Determine UseGlobal
	useGlobal := false
	if manifest.UseGlobal != nil {
		useGlobal = *manifest.UseGlobal
	} else {
		// Infer: does internal/app/config.go exist?
		if _, err := os.Stat(filepath.Join(dir, "internal", "app", "config.go")); err == nil {
			useGlobal = true
		}
	}

	// Determine IncludeExample
	includeExample := false
	if _, err := os.Stat(filepath.Join(dir, "internal", "commands", "greet.go")); err == nil {
		includeExample = true
	}

	// Merge current + requested integrations
	merged := mergeIntegrations(current, toAdd)

	data := baseTemplateData(name, module, useGlobal, includeExample)
	data.UseSQLite = merged.SQLite
	data.UsePostgres = merged.Postgres
	data.UseAI = merged.AI

	return data, nil
}


func renderIntegrationFiles(data TemplateData) (map[string][]byte, error) {
	result := make(map[string][]byte)

	// The two integration-affected templates
	templatePaths := []struct {
		tmpl    string
		outPath string
	}{
		{"templates/structured/internal/app/context.go.tmpl", "internal/app/context.go"},
		{"templates/structured/internal/commands/root.go.tmpl", "internal/commands/root.go"},
	}

	for _, tp := range templatePaths {
		rendered, err := renderTemplate(templates, tp.tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", tp.outPath, err)
		}

		// Skip files that render to empty (e.g., context.go when no integrations)
		if len(bytes.TrimSpace(rendered)) == 0 {
			continue
		}

		result[tp.outPath] = rendered
	}

	return result, nil
}

func checkFileSafety(dir, relPath string, manifest *scaffoldManifest) fileSafetyResult {
	fullPath := filepath.Join(dir, relPath)

	existing, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fileSafetyCreate
		}
		return fileSafetyConflict // can't read = treat as conflict
	}

	// Find the file in the manifest
	currentHash := sha256Hex(existing)
	for _, f := range manifest.Files {
		if f.Path == relPath {
			if f.ContentSHA256 == currentHash {
				return fileSafetySafe
			}
			return fileSafetyConflict
		}
	}

	// File exists but not in manifest — treat as conflict
	return fileSafetyConflict
}

func addGoDependencies(dir string, integrations []string, verbose bool) error {
	var packages []string
	for _, name := range integrations {
		dep, ok := integrationDeps[name]
		if !ok {
			continue
		}
		packages = append(packages, dep.Packages...)
	}

	if len(packages) == 0 {
		return nil
	}

	ctx := context.Background()

	goGetArgs := append([]string{"get"}, packages...)
	if err := runCommand(ctx, dir, verbose, "go", goGetArgs...); err != nil {
		return fmt.Errorf("go get: %w", err)
	}

	if err := runCommand(ctx, dir, verbose, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}

func updateAddManifest(dir string, manifest *scaffoldManifest, data TemplateData, renderedFiles map[string][]byte) error {
	// Derive merged integrations directly from template data
	merged := &manifestIntegrations{
		SQLite:   data.UseSQLite,
		Postgres: data.UsePostgres,
		AI:       data.UseAI,
	}

	// Update manifest to v2
	manifest.Version = scaffoldManifestV2
	manifest.Integrations = merged
	if manifest.Module == "" {
		manifest.Module = data.Module
	}
	if manifest.Mode == "" {
		manifest.Mode = "structured"
	}
	if manifest.UseGlobal == nil {
		manifest.UseGlobal = boolPtr(data.UseGlobal)
	}

	// Update file hashes for re-rendered files
	for relPath, content := range renderedFiles {
		hash := sha256Hex(content)
		found := false
		for i, f := range manifest.Files {
			if f.Path == relPath {
				manifest.Files[i].ContentSHA256 = hash
				manifest.Files[i].TemplateSHA256 = hash
				manifest.Files[i].Action = "overwrite"
				found = true
				break
			}
		}
		if !found {
			manifest.Files = append(manifest.Files, scaffoldManifestFile{
				Path:           relPath,
				Action:         "create",
				TemplateSHA256: hash,
				ContentSHA256:  hash,
				Mode:           0644,
			})
		}
	}

	// Sort files for deterministic output
	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})

	// Write manifest
	now := time.Now().UTC()
	manifest.GeneratedAt = &now

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')

	return writeScaffoldManifest(dir, manifestData)
}

func failAddCommand(err error, jsonOutput bool, output *addCommandOutput, code commandErrorCode, exitCode int) error {
	cmdErr := &commandError{
		Err:      err,
		Code:     code,
		Outcome:  commandOutcomeFailure,
		ExitCode: exitCode,
	}

	if jsonOutput && output != nil {
		output.Outcome = commandOutcomeFailure
		output.ErrorCode = code
		output.ExitCode = exitCode
		output.Error = err.Error()
		_ = emitJSON(output)
	}

	return cmdErr
}
