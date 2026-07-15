package generator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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
		err := s.verify(ctx, req.TargetDir, runtime, verificationPlan{Timeout: req.VerifyTimeout, Tidy: true, Build: true}, &result.Checks)
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

	if req.DryRun {
		return s.completeDryRun(ctx, req, runtime, result)
	}

	if err := s.resolveDependencies(ctx, req, runtime, &result.Checks); err != nil {
		recovery := s.dependencyRecoveryCommand(req)
		return result, &OperationError{Kind: ErrorScaffoldFailed, Partial: true,
			Err: fmt.Errorf("dependency preparation failed; generated files were kept; recover with: %s: %w", recovery, err)}
	}
	if req.Verify {
		result.VerifyRan = true
		verifyErr := s.verify(ctx, req.TargetDir, runtime, verificationPlan{Timeout: req.VerifyTimeout, Build: true}, &result.Checks)
		result.VerifyPassed = verifyErr == nil
		if verifyErr != nil {
			return result, &OperationError{Kind: ErrorVerifyFailed, Partial: true,
				Err: fmt.Errorf("project generated at %s, but verification failed: %w", req.TargetDir, verifyErr)}
		}
	}
	result.NextDir = req.TargetDir
	result.NextCommand = "go"
	if req.Mode == modeFlat {
		result.NextArgs = []string{"build", "-o", req.ProjectName, "."}
	} else {
		result.NextArgs = []string{"build", "-o", req.ProjectName, "./cmd"}
	}
	result.NextSteps = nextSteps(req, result.NextArgs)
	return result, nil
}

func (s *Service) completeDryRun(ctx context.Context, req newRequest, runtime Runtime, result CreateResult) (CreateResult, error) {
	dependencyErr, verifyErr := s.prepareDryRun(ctx, req, runtime, &result.Checks)
	if dependencyErr != nil {
		return result, &OperationError{Kind: ErrorScaffoldFailed, Err: dependencyErr}
	}
	if !req.Verify {
		return result, nil
	}
	result.VerifyRan = true
	result.VerifyPassed = verifyErr == nil
	if verifyErr != nil {
		return result, &OperationError{Kind: ErrorVerifyFailed, Err: verifyErr}
	}
	return result, nil
}

func createResultFromRequest(req newRequest) CreateResult {
	return CreateResult{
		Preset: req.Preset, Mode: req.Mode, ProjectName: req.ProjectName,
		TargetDir: req.TargetDir, Module: req.Module, ConfigScope: req.ConfigScope,
		UseGlobal: req.UseGlobal, DryRun: req.DryRun, WriteManifest: req.WriteManifest,
		VerifyRequested: req.Verify, VerifyOnly: req.VerifyOnly, IncludeExample: req.IncludeExample,
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

func (s *Service) resolveDependencies(ctx context.Context, req newRequest, runtime Runtime, checks *[]CheckResult) error {
	packages := s.dependencyPackages(req)
	if err := s.runCheckedGoCommand(ctx, req.TargetDir, runtime, checks, append([]string{"get"}, packages...)...); err != nil {
		return fmt.Errorf("go get: %w", err)
	}
	if err := s.runCheckedGoCommand(ctx, req.TargetDir, runtime, checks, "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	return nil
}

func (s *Service) dependencyPackages(req newRequest) []string {
	packages := []string{"github.com/alecthomas/kong@" + defaultKongVersion}
	if req.UseGlobal {
		packages = append(packages, "github.com/dotcommander/gokart@"+defaultGokartVersion)
	}
	packages = append(packages, integrationDependencies(templateDataForRequest(req))...)
	sort.Strings(packages)
	return packages
}

func (s *Service) dependencyRecoveryCommand(req newRequest) string {
	return "cd " + shellQuote(req.TargetDir) + " && go get " + strings.Join(s.dependencyPackages(req), " ") + " && go mod tidy"
}

func nextSteps(req newRequest, buildArgs []string) []string {
	steps := []string{"cd " + shellQuote(req.DisplayDir)}
	if req.IncludeExample {
		if req.Mode == modeFlat {
			steps = append(steps, "go run . greet --name World")
		} else {
			steps = append(steps, "go run ./cmd greet --name World")
		}
	}
	steps = append(steps, "go "+strings.Join(buildArgs, " "))
	return steps
}

func templateDataForRequest(req newRequest) TemplateData {
	return TemplateData{UseSQLite: req.UseSQLite, UsePostgres: req.UsePostgres, UseAI: req.UseAI, UseRedis: req.UseRedis}
}
