package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	integrationRedis    = "redis"
)

var validIntegrations = map[string]bool{
	integrationSQLite:   true,
	integrationPostgres: true,
	integrationAI:       true,
	integrationRedis:    true,
}

type integrationDep struct {
	Packages []string
}

var integrationDeps = map[string]integrationDep{
	integrationSQLite:   {Packages: []string{"github.com/dotcommander/gokart/sqlite@" + defaultGokartSQLiteVersion}},
	integrationPostgres: {Packages: []string{"github.com/dotcommander/gokart/postgres@" + defaultGokartPostgresVersion, "github.com/jackc/pgx/v5@latest"}},
	integrationAI:       {Packages: []string{"github.com/dotcommander/gokart/ai@" + defaultGokartAIVersion, "github.com/openai/openai-go/v3@latest"}},
	integrationRedis:    {Packages: []string{"github.com/dotcommander/gokart/cache@" + defaultGokartCacheVersion, "github.com/redis/go-redis/v9@" + defaultRedisVersion}},
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

type addFlowError struct {
	Err      error
	Code     commandErrorCode
	ExitCode int
}

func (e *addFlowError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *addFlowError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <integration>...",
		Short: "Add integrations to an existing GoKart project",
		Long:  "Add SQLite, PostgreSQL, Redis, or OpenAI integrations to an existing structured project.\nRe-renders only integration-affected files (context.go, root.go) and runs go get.",
		Example: `  gokart add sqlite
  gokart add ai
  gokart add sqlite ai
  gokart add postgres --dry-run
  gokart add ai --force
  gokart add ai --json
  gokart add ai --verify`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("specify at least one integration: sqlite, postgres, ai, redis\n\nExamples:\n  gokart add sqlite\n  gokart add ai\n  gokart add sqlite ai postgres")
			}
			return nil
		},
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
	configureJSONCommand(cmd, jsonOutput)

	integrations, err := collectAddIntegrations(args)
	output := addCommandOutput{Outcome: commandOutcomeSuccess}
	if err != nil {
		return emitCommandError(err, jsonOutput, &output, commandFailureInfo{Code: errorCodeInvalidArguments, Outcome: commandOutcomeFailure, ExitCode: exitCodeInvalidArguments})
	}

	req, err := buildAddRequest(cmd, integrations)
	if err != nil {
		output.DryRun = false
		output.Integrations = append([]string(nil), integrations...)
		return emitCommandError(err, jsonOutput, &output, commandFailureInfo{Code: errorCodeScaffoldFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeScaffoldFailed})
	}

	output.DryRun = req.DryRun
	output.Integrations = append([]string(nil), req.Integrations...)
	output.VerifyRequested = req.Verify

	plan, err := planAddChanges(req, &output)
	if err != nil {
		var flowErr *addFlowError
		if errors.As(err, &flowErr) {
			return emitCommandError(err, jsonOutput, &output, commandFailureInfo{Code: flowErr.Code, Outcome: commandOutcomeFailure, ExitCode: flowErr.ExitCode})
		}
		return emitCommandError(err, jsonOutput, &output, commandFailureInfo{Code: errorCodeScaffoldFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeScaffoldFailed})
	}

	if len(plan.ToAdd) == 0 {
		if !jsonOutput {
			for _, name := range output.AlreadyPresent {
				cli.Warning("%s already enabled", name)
			}
		}
		if jsonOutput {
			_ = emitJSON(output)
		}
		return nil
	}

	if req.DryRun {
		printAddResult(req, output)
		if jsonOutput {
			_ = emitJSON(output)
		}
		return nil
	}

	if err := applyAddChanges(req, plan, &output); err != nil {
		var flowErr *addFlowError
		if errors.As(err, &flowErr) {
			return emitCommandError(err, jsonOutput, &output, commandFailureInfo{Code: flowErr.Code, Outcome: commandOutcomeFailure, ExitCode: flowErr.ExitCode})
		}
		return emitCommandError(err, jsonOutput, &output, commandFailureInfo{Code: errorCodeScaffoldFailed, Outcome: commandOutcomeFailure, ExitCode: exitCodeScaffoldFailed})
	}

	if jsonOutput {
		_ = emitJSON(output)
	}

	return nil
}

func isFlatProject(manifest *scaffoldManifest) bool {
	return manifest.TemplateRoot == "templates/flat" || manifest.Mode == modeFlat
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
	if strings.Contains(content, "gokart/cache") {
		result.Redis = true
	}
	return module, result
}

func integrationEnabled(current *manifestIntegrations, name string) bool {
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
	case integrationRedis:
		return current.Redis
	default:
		return false
	}
}

func setIntegration(m *manifestIntegrations, name string, enable bool) {
	if m == nil {
		return
	}
	switch name {
	case integrationSQLite:
		m.SQLite = enable
	case integrationPostgres:
		m.Postgres = enable
	case integrationAI:
		m.AI = enable
	case integrationRedis:
		m.Redis = enable
	}
}

func mergeIntegrations(current *manifestIntegrations, toAdd []string) *manifestIntegrations {
	merged := &manifestIntegrations{}
	if current != nil {
		*merged = *current
	}
	for _, name := range toAdd {
		setIntegration(merged, name, true)
	}
	return merged
}

func inferTemplateData(manifest *scaffoldManifest, dir string, current *manifestIntegrations, toAdd []string, goModModule string) (TemplateData, error) {
	// Get module from manifest or go.mod
	module := manifest.Module
	if module == "" {
		module = goModModule
		if module == "" {
			return TemplateData{}, errors.New("no module found in manifest or go.mod")
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
	data.UseRedis = merged.Redis

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
		Redis:    data.UseRedis,
	}

	// Update manifest to v2
	manifest.Version = scaffoldManifestV2
	manifest.Integrations = merged
	if manifest.Module == "" {
		manifest.Module = data.Module
	}
	if manifest.Mode == "" {
		manifest.Mode = modeStructured
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
	slices.SortFunc(manifest.Files, func(a, b scaffoldManifestFile) int {
		return strings.Compare(a.Path, b.Path)
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
