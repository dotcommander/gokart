package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type newRequest struct {
	Preset             string
	Mode               string
	ProjectName        string
	TargetDir          string
	Module             string
	ConfigScope        string
	UseSQLite          bool
	UsePostgres        bool
	UseAI              bool
	UseRedis           bool
	IncludeExample     bool
	UseGlobal          bool
	DryRun             bool
	WriteManifest      bool
	Verify             bool
	VerifyOnly         bool
	VerifyTimeout      time.Duration
	ExistingFilePolicy ExistingFilePolicy
	Warnings           []string
}

//nolint:gocyclo,funlen // flag validation, complexity is inherent
func buildNewRequest(cmd *cobra.Command, args []string) (newRequest, error) {
	var req newRequest

	preset, projectArg, err := parseNewInvocation(args)
	if err != nil {
		return req, err
	}
	if preset != defaultPreset {
		return req, fmt.Errorf("unsupported preset %q", preset)
	}
	req.Preset = preset

	flat, _ := cmd.Flags().GetBool(newFlagFlat)
	module, _ := cmd.Flags().GetString(newFlagModule)
	req.UseSQLite, _ = cmd.Flags().GetBool(newFlagSQLite)
	req.UsePostgres, _ = cmd.Flags().GetBool(newFlagPostgres)
	req.UseAI, _ = cmd.Flags().GetBool(newFlagAI)
	req.UseRedis, _ = cmd.Flags().GetBool(newFlagRedis)
	req.IncludeExample, _ = cmd.Flags().GetBool(newFlagExample)
	local, _ := cmd.Flags().GetBool(newFlagLocal)
	global, _ := cmd.Flags().GetBool(newFlagGlobal)
	req.ConfigScope, _ = cmd.Flags().GetString(newFlagConfigScope)
	req.DryRun, _ = cmd.Flags().GetBool(newFlagDryRun)
	force, _ := cmd.Flags().GetBool(newFlagForce)
	skipExisting, _ := cmd.Flags().GetBool(newFlagSkipExisting)
	noManifest, _ := cmd.Flags().GetBool(newFlagNoManifest)
	req.WriteManifest = !noManifest
	req.Verify, _ = cmd.Flags().GetBool(newFlagVerify)
	req.VerifyOnly, _ = cmd.Flags().GetBool(newFlagVerifyOnly)
	req.VerifyTimeout, _ = cmd.Flags().GetDuration(newFlagVerifyTimeout)

	if req.VerifyTimeout < 0 {
		return req, fmt.Errorf("invalid --verify-timeout %s (must be >= 0)", req.VerifyTimeout)
	}

	if flat {
		req.Mode = modeFlat
	} else {
		req.Mode = modeStructured
	}

	projectName, targetDir, err := normalizeProjectArg(projectArg)
	if err != nil {
		return req, err
	}
	req.ProjectName = projectName
	req.TargetDir = targetDir

	if module == "" {
		module = projectName
	}
	req.Module = module

	if req.VerifyOnly { //nolint:nestif // verify-only branch validates multiple flags, nesting is inherent
		if req.DryRun {
			return req, errors.New("cannot combine --verify-only with --dry-run")
		}

		req.Verify = true
		req.ExistingFilePolicy = ExistingFilePolicyFail
		useGlobal, _, resolveErr := resolveUseGlobal(flat, false, false, configScopeAuto)
		if resolveErr != nil {
			return req, resolveErr
		}
		req.UseGlobal = useGlobal

		if ignored := verifyOnlyIgnoredFlags(cmd); len(ignored) > 0 {
			req.Warnings = append(req.Warnings, "--verify-only ignores generation flags: "+strings.Join(ignored, ", "))
		}

		if err := requireExistingTargetDir(targetDir); err != nil {
			return req, err
		}

		return req, nil
	}

	useGlobal, warnings, err := resolveUseGlobal(flat, local, global, req.ConfigScope)
	if err != nil {
		return req, err
	}
	req.UseGlobal = useGlobal
	req.Warnings = append(req.Warnings, warnings...)

	existingPolicy, err := resolveExistingFilePolicy(force, skipExisting)
	if err != nil {
		return req, err
	}
	req.ExistingFilePolicy = existingPolicy

	if err := validateModulePath(module); err != nil {
		return req, fmt.Errorf("invalid module path %q: %w", module, err)
	}

	if err := validateTargetDir(targetDir); err != nil {
		return req, err
	}

	return req, nil
}

func verifyOnlyIgnoredFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}

	ignored := make([]string, 0, len(verifyOnlyIgnoredFlagNames))
	for _, name := range verifyOnlyIgnoredFlagNames {
		if cmd.Flags().Changed(name) {
			ignored = append(ignored, "--"+name)
		}
	}

	return ignored
}

func newCommandOutputFromRequest(req newRequest) newCommandOutput {
	return newCommandOutput{
		Outcome:            commandOutcomeSuccess,
		ExitCode:           0,
		Preset:             req.Preset,
		Mode:               req.Mode,
		ProjectName:        req.ProjectName,
		TargetDir:          req.TargetDir,
		Module:             req.Module,
		ConfigScope:        req.ConfigScope,
		UseGlobal:          req.UseGlobal,
		DryRun:             req.DryRun,
		WriteManifest:      req.WriteManifest,
		VerifyRequested:    req.Verify,
		VerifyOnly:         req.VerifyOnly,
		ExistingFilePolicy: req.ExistingFilePolicy,
		Warnings:           append([]string(nil), req.Warnings...),
	}
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
		return false, nil, errors.New("cannot use --local and --global together")
	}

	scope := strings.ToLower(strings.TrimSpace(configScope))
	if scope == "" {
		scope = configScopeAuto
	}

	if scope != configScopeAuto && (local || global) {
		return false, nil, errors.New("cannot combine --config-scope with --local or --global")
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
		return "", errors.New("cannot use --force and --skip-existing together")
	}

	if force {
		return ExistingFilePolicyOverwrite, nil
	}

	if skipExisting {
		return ExistingFilePolicySkip, nil
	}

	return ExistingFilePolicyFail, nil
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

func requireExistingTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("target directory %q does not exist (required for --verify-only)", targetDir)
		}
		return fmt.Errorf("check target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path %q exists and is not a directory", targetDir)
	}

	return nil
}
