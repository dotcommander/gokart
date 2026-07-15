package generator

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

func (s *Service) Create(ctx context.Context, request CreateRequest, runtime Runtime) (CreateResult, error) {
	request.lookupEnv = s.deps.LookupEnv
	req, err := buildNewRequest(&request)
	if err != nil {
		return CreateResult{}, &OperationError{Kind: ErrorInvalidArguments, Err: err}
	}
	result := createResultFromRequest(req)
	if req.VerifyOnly {
		result.VerifyRan = true
		err := s.verify(ctx, req.TargetDir, req.VerifyTimeout, runtime)
		result.VerifyPassed = err == nil
		if err != nil {
			return result, &OperationError{Kind: ErrorVerifyFailed, Err: fmt.Errorf("verification failed for %s: %w", req.TargetDir, err)}
		}
		return result, nil
	}

	runtime.report(EventScaffoldStart, fmt.Sprintf("%s %s project (%s preset): %s", scaffoldVerb(req.DryRun), req.Mode, req.Preset, req.ProjectName))
	applyResult, err := scaffoldProject(req, newApplyOptions(req), s.deps.GeneratorVersion)
	if err != nil {
		return result, classifyScaffoldError(err)
	}
	result.Result = applyResult

	if !req.DryRun {
		if err := s.resolveDependencies(ctx, req, runtime); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("dependency resolution failed: %v", err))
		}
	}
	if req.Verify {
		result.VerifyRan = true
		verifyErr := s.verifyCreate(ctx, req, runtime)
		result.VerifyPassed = verifyErr == nil
		if verifyErr != nil {
			partial := !req.DryRun
			return result, &OperationError{Kind: ErrorVerifyFailed, Partial: partial, Err: verifyErr}
		}
	}
	if !req.DryRun {
		result.NextDir = req.TargetDir
		result.NextCommand = "go"
		result.NextArgs = []string{"build", "./..."}
	}
	return result, nil
}

func createResultFromRequest(req newRequest) CreateResult {
	return CreateResult{
		Preset: req.Preset, Mode: req.Mode, ProjectName: req.ProjectName,
		TargetDir: req.TargetDir, Module: req.Module, ConfigScope: req.ConfigScope,
		UseGlobal: req.UseGlobal, DryRun: req.DryRun, WriteManifest: req.WriteManifest,
		VerifyRequested: req.Verify, VerifyOnly: req.VerifyOnly,
		ExistingFilePolicy: req.ExistingFilePolicy, Warnings: append([]string(nil), req.Warnings...),
	}
}

func scaffoldVerb(dryRun bool) string {
	if dryRun {
		return "Dry run: planning"
	}
	return "Scaffolding"
}

func classifyScaffoldError(err error) error {
	var conflict *ExistingFileConflictError
	if errors.As(err, &conflict) {
		return &OperationError{Kind: ErrorExistingFileConflict, Conflicts: append([]string(nil), conflict.Paths...), Err: err}
	}
	var lock *ApplyLockError
	if errors.As(err, &lock) {
		return &OperationError{Kind: ErrorTargetLocked, Err: err}
	}
	return &OperationError{Kind: ErrorScaffoldFailed, Err: err}
}

func newApplyOptions(req newRequest) ApplyOptions {
	return ApplyOptions{DryRun: req.DryRun, ExistingFilePolicy: req.ExistingFilePolicy, SkipManifest: !req.WriteManifest}
}

func scaffoldProject(req newRequest, opts ApplyOptions, version string) (*ApplyResult, error) {
	opts.GeneratorVersion = version
	switch req.Mode {
	case modeFlat:
		return ScaffoldFlat(req.TargetDir, req.ProjectName, req.Module, req.UseGlobal, req.IncludeExample, opts)
	case modeStructured:
		return ScaffoldStructured(req.TargetDir, req.ProjectName, req.Module, req.UseSQLite, req.UsePostgres, req.UseAI, req.UseRedis, req.UseGlobal, req.IncludeExample, opts)
	default:
		return nil, fmt.Errorf("unsupported mode %q", req.Mode)
	}
}

func (s *Service) resolveDependencies(ctx context.Context, req newRequest, runtime Runtime) error {
	packages := []string{"github.com/alecthomas/kong@" + defaultKongVersion}
	if req.UseGlobal {
		packages = append(packages, "github.com/dotcommander/gokart@"+defaultGokartVersion)
	}
	packages = append(packages, integrationDependencies(templateDataForRequest(req))...)
	sort.Strings(packages)
	if err := s.runGoCommand(ctx, req.TargetDir, runtime, append([]string{"get"}, packages...)...); err != nil {
		return fmt.Errorf("go get: %w", err)
	}
	if err := s.runGoCommand(ctx, req.TargetDir, runtime, "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	return nil
}

func templateDataForRequest(req newRequest) TemplateData {
	return TemplateData{UseSQLite: req.UseSQLite, UsePostgres: req.UsePostgres, UseAI: req.UseAI, UseRedis: req.UseRedis}
}
