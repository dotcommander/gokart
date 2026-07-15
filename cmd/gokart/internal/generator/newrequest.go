package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type newRequest struct {
	Preset             string
	Mode               string
	ProjectName        string
	TargetDir          string
	DisplayDir         string
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

func buildNewRequest(cmd *CreateRequest, args ...[]string) (newRequest, error) {
	if len(args) > 0 {
		cmd.Args = args[0]
	}
	preset, projectArg, err := parseNewInvocation(cmd.Args)
	if err != nil {
		return newRequest{}, err
	}
	if preset != defaultPreset {
		return newRequest{}, fmt.Errorf("unsupported preset %q", preset)
	}
	req, err := newRequestFromCommand(cmd, preset)
	if err != nil {
		return req, err
	}
	projectName, targetDir, err := normalizeProjectArg(projectArg)
	if err != nil {
		return req, err
	}
	req.ProjectName = projectName
	req.DisplayDir = filepath.Clean(targetDir)
	if !filepath.IsAbs(targetDir) && cmd.WorkingDir != "" {
		targetDir = filepath.Join(cmd.WorkingDir, targetDir)
	}
	req.TargetDir = filepath.Clean(targetDir)
	req.Module = cmd.Module
	if req.Module == "" {
		req.Module = projectName
	}

	if req.VerifyOnly {
		return finishVerifyOnlyRequest(req, cmd)
	}
	return finishGenerationRequest(req, cmd)
}

func newRequestFromCommand(cmd *CreateRequest, preset string) (newRequest, error) {
	useSQLite, usePostgres, err := resolveDB(cmd.DB)
	if err != nil {
		return newRequest{}, err
	}
	if cmd.Verify && cmd.NoVerify {
		return newRequest{}, errors.New("cannot use --verify and --no-verify together")
	}
	if cmd.VerifyTimeout < 0 {
		return newRequest{}, fmt.Errorf("invalid --verify-timeout %s (must be >= 0)", cmd.VerifyTimeout)
	}
	req := newRequest{
		Preset: preset, UseSQLite: useSQLite, UsePostgres: usePostgres, UseAI: cmd.AI,
		UseRedis: cmd.Redis, IncludeExample: cmd.Example, ConfigScope: cmd.ConfigScope,
		DryRun: cmd.DryRun, Verify: cmd.Verify, VerifyOnly: cmd.VerifyOnly,
		VerifyTimeout: cmd.VerifyTimeout,
	}
	if err := resolveRequestMode(&req, cmd.Flat, cmd.Structured); err != nil {
		return newRequest{}, err
	}
	return req, nil
}

func resolveRequestMode(req *newRequest, flat, structured bool) error {
	if req.VerifyOnly {
		return nil
	}
	if flat && structured {
		return errors.New("cannot use --flat and --structured together")
	}
	hasIntegration := req.UseSQLite || req.UsePostgres || req.UseAI || req.UseRedis
	if flat && hasIntegration {
		return errors.New("integrations (--db sqlite/postgres, --ai, --redis) require structured mode; remove --flat")
	}
	if structured || hasIntegration {
		req.Mode = modeStructured
	} else {
		req.Mode = modeFlat
	}
	return nil
}

func finishVerifyOnlyRequest(req newRequest, cmd *CreateRequest) (newRequest, error) {
	if req.DryRun {
		return req, errors.New("cannot combine --verify-only with --dry-run")
	}
	req.Verify = true
	req.ExistingFilePolicy = ExistingFilePolicyFail
	if ignored := verifyOnlyIgnoredFlags(cmd); len(ignored) > 0 {
		req.Warnings = append(req.Warnings, "--verify-only ignores generation flags: "+strings.Join(ignored, ", "))
	}
	if err := requireExistingTargetDir(req.TargetDir); err != nil {
		return req, err
	}
	return req, nil
}

func finishGenerationRequest(req newRequest, cmd *CreateRequest) (newRequest, error) {
	useGlobal, warnings, err := resolveUseGlobal(cmd.Local, cmd.Global, req.ConfigScope)
	if err != nil {
		return req, err
	}
	req.UseGlobal = useGlobal
	req.Warnings = append(req.Warnings, warnings...)
	if req.Mode == modeFlat && cmd.NoManifest {
		req.Warnings = append(req.Warnings, "--no-manifest has no effect: flat projects are already unmanaged")
	}
	req.WriteManifest = resolveWriteManifest(req, cmd.NoManifest)
	req.ExistingFilePolicy, err = resolveExistingFilePolicy(cmd.Force, cmd.SkipExisting)
	if err != nil {
		return req, err
	}
	if err := validateModulePath(req.Module); err != nil {
		return req, fmt.Errorf("invalid module path %q: %w", req.Module, err)
	}
	if err := validateTargetDir(req.TargetDir, req.ExistingFilePolicy); err != nil {
		return req, err
	}
	req.Verify = resolveAutoVerify(req, cmd.Verify, cmd.NoVerify, cmd.lookupEnv, &req.Warnings)
	return req, nil
}

func resolveWriteManifest(req newRequest, noManifest bool) bool {
	return req.Mode == modeStructured && !noManifest
}

func verifyOnlyIgnoredFlags(cmd *CreateRequest) []string {
	if cmd == nil {
		return nil
	}

	ignored := make([]string, 0, len(verifyOnlyIgnoredFlagNames))
	for _, name := range verifyOnlyIgnoredFlagNames {
		if cmd.Changed[name] {
			ignored = append(ignored, "--"+name)
		}
	}

	return ignored
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

func resolveUseGlobal(local, global bool, configScope string) (bool, []string, error) {
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
		if local {
			warnings = append(warnings, "--local is already the default")
		}
		return global, warnings, nil
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

func validateTargetDir(targetDir string, policy ExistingFilePolicy) error {
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

	if policy == ExistingFilePolicyFail {
		entries, readErr := os.ReadDir(targetDir)
		if readErr != nil {
			return fmt.Errorf("read target directory: %w", readErr)
		}
		if len(entries) > 0 {
			return fmt.Errorf("directory %q already exists and is not empty (use --force to overwrite)", targetDir)
		}
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
