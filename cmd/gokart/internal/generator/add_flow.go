package generator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type addRequest struct {
	Dir           string
	Integrations  []string
	DryRun        bool
	Force         bool
	Verify        bool
	VerifyTimeout time.Duration
}

type addPlan struct {
	Manifest      *scaffoldManifest
	ToAdd         []string
	Data          TemplateData
	RenderedFiles map[string][]byte
	RenderedPaths []string
}

func (s *Service) Add(ctx context.Context, request AddRequest, runtime Runtime) (AddResult, error) {
	integrations, err := collectAddIntegrations(request.Integrations)
	if err != nil || len(integrations) == 0 {
		if err == nil {
			err = errors.New("specify at least one integration: sqlite, postgres, ai, redis")
		}
		return AddResult{}, &OperationError{Kind: ErrorInvalidArguments, Err: err}
	}
	req := addRequest{
		Dir: request.Dir, Integrations: integrations, DryRun: request.DryRun,
		Force: request.Force, Verify: request.Verify, VerifyTimeout: request.VerifyTimeout,
	}
	result := AddResult{Integrations: append([]string(nil), integrations...), DryRun: req.DryRun, VerifyRequested: req.Verify}
	var plan *addPlan
	mutate := func() error {
		plan, err = planAddChanges(req, &result, s.deps.GeneratorVersion)
		if err != nil || req.DryRun || len(plan.ToAdd) == 0 {
			return err
		}
		return s.applyAddChanges(ctx, req, plan, &result, runtime)
	}
	if req.DryRun {
		err = mutate()
	} else {
		err = withTargetMutationLock(req.Dir, mutate)
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func wrapAddFlowError(err error, kind ErrorKind) error {
	if err == nil {
		return nil
	}
	return &OperationError{Kind: kind, Err: err}
}

func collectAddIntegrations(args []string) ([]string, error) {
	seen := make(map[string]bool, len(args))
	requested := make([]string, 0, len(args))
	for _, arg := range args {
		name := strings.ToLower(strings.TrimSpace(arg))
		if _, ok := integrationRegistry[name]; !ok {
			return nil, fmt.Errorf("unknown integration: %s (valid: sqlite, postgres, ai, redis)", name)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		requested = append(requested, name)
	}

	return requested, nil
}

func planAddChanges(req addRequest, output *addCommandOutput, generatorVersion ...string) (*addPlan, error) {
	if _, err := os.Stat(filepath.Join(req.Dir, scaffoldManifestPath)); os.IsNotExist(err) && looksLikeFlatProject(req.Dir) {
		return nil, flatAddUnsupportedError()
	}
	manifest, err := readAddManifest(req.Dir)
	if err != nil {
		return nil, wrapAddFlowError(err, ErrorManifestNotFound)
	}
	if generatorVersionSkewed(manifest, generatorVersion) {
		output.Warnings = append(output.Warnings, fmt.Sprintf("project was scaffolded by gokart %s but you are running %s; templates may not match the original project layout", manifest.GeneratorVersion, generatorVersion[0]))
	}

	// Warn (non-fatal) on template-version skew: a project scaffolded by an
	// older gokart may not match templates rendered by the running version.
	if isFlatProject(manifest) {
		return nil, flatAddUnsupportedError()
	}

	goModModule, current := detectCurrentIntegrations(manifest, req.Dir)

	toAdd := classifyRequestedIntegrations(req.Integrations, current, output)
	output.Added = append(output.Added[:0], toAdd...)

	if len(toAdd) == 0 {
		return &addPlan{Manifest: manifest}, nil
	}

	data, err := inferTemplateData(manifest, req.Dir, current, toAdd, goModModule)
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("infer project state: %w", err), ErrorScaffoldFailed)
	}

	renderedFiles, err := renderIntegrationFiles(data)
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("render templates: %w", err), ErrorScaffoldFailed)
	}

	renderedPaths := make([]string, 0, len(renderedFiles))
	for relPath := range renderedFiles {
		renderedPaths = append(renderedPaths, relPath)
	}
	sort.Strings(renderedPaths)

	if err := classifyRenderedFiles(req, manifest, renderedPaths, output); err != nil {
		return nil, err
	}

	return &addPlan{
		Manifest:      manifest,
		ToAdd:         toAdd,
		Data:          data,
		RenderedFiles: renderedFiles,
		RenderedPaths: renderedPaths,
	}, nil
}

func looksLikeFlatProject(dir string) bool {
	_, mainErr := os.Stat(filepath.Join(dir, "main.go"))
	_, modErr := os.Stat(filepath.Join(dir, "go.mod"))
	_, cmdErr := os.Stat(filepath.Join(dir, "cmd", "main.go"))
	return mainErr == nil && modErr == nil && os.IsNotExist(cmdErr)
}

func flatAddUnsupportedError() error {
	return wrapAddFlowError(errors.New("gokart add requires a managed structured project; add the relevant GoKart package to this flat project manually, or start future growable projects with `gokart new <name> --structured`"), ErrorFlatModeUnsupported)
}

func generatorVersionSkewed(manifest *scaffoldManifest, versions []string) bool {
	return len(versions) > 0 && manifest.GeneratorVersion != "" && manifest.GeneratorVersion != versions[0]
}

func classifyRequestedIntegrations(requested []string, current *manifestIntegrations, output *addCommandOutput) []string {
	toAdd := make([]string, 0, len(requested))
	for _, name := range requested {
		if integrationEnabled(current, name) {
			output.AlreadyPresent = append(output.AlreadyPresent, name)
			continue
		}
		toAdd = append(toAdd, name)
	}
	return toAdd
}

func classifyRenderedFiles(req addRequest, manifest *scaffoldManifest, paths []string, output *addCommandOutput) error {
	for _, relPath := range paths {
		switch checkFileSafety(req.Dir, relPath, manifest) {
		case fileSafetyCreate:
			output.FilesCreated = append(output.FilesCreated, relPath)
		case fileSafetySafe:
			output.FilesOverwritten = append(output.FilesOverwritten, relPath)
		case fileSafetyConflict:
			if !req.Force {
				return &OperationError{Kind: ErrorExistingFileConflict, Conflicts: []string{relPath}, Err: fmt.Errorf("file %s has been modified (use --force to overwrite)", relPath)}
			}
			output.FilesOverwritten = append(output.FilesOverwritten, relPath)
			output.Warnings = append(output.Warnings, "force-overwriting modified file: "+relPath)
		}
	}
	return nil
}

func (s *Service) applyAddChanges(ctx context.Context, req addRequest, plan *addPlan, output *addCommandOutput, runtime Runtime) error {
	journal, applied, err := applyAddFileWrites(req, plan)
	if err != nil {
		return err
	}

	depActions, err := journalDependencyFiles(req.Dir, journal)
	if err != nil {
		err := rollbackWithError(fmt.Errorf("prepare dependency rollback: %w", err), applied, journal)
		return wrapAddFlowError(err, ErrorScaffoldFailed)
	}

	// From here on, any failure must revert every file written above.
	depErr := s.addGoDependencies(ctx, req.Dir, plan.ToAdd, runtime, &output.Checks)
	applied, err = markDependencyFilesApplied(depActions, journal, applied)
	if depErr != nil {
		if err != nil {
			depErr = errors.Join(depErr, fmt.Errorf("record dependency rollback state: %w", err))
		}
		err := rollbackWithError(fmt.Errorf("add dependencies: %w", depErr), applied, journal)
		return wrapAddFlowError(err, ErrorScaffoldFailed)
	}
	if err != nil {
		err := rollbackWithError(fmt.Errorf("record dependency rollback state: %w", err), applied, journal)
		return wrapAddFlowError(err, ErrorScaffoldFailed)
	}

	if mfErr := updateAddManifest(req.Dir, plan.Manifest, plan.Data, plan.RenderedFiles); mfErr != nil {
		err := rollbackWithError(fmt.Errorf("update manifest: %w", mfErr), applied, journal)
		return wrapAddFlowError(err, ErrorScaffoldFailed)
	}

	if err := journal.markCompleted(); err != nil {
		return wrapAddFlowError(fmt.Errorf("finalize add journal: %w", err), ErrorScaffoldFailed)
	}
	if err := journal.cleanup(); err != nil {
		return wrapAddFlowError(fmt.Errorf("cleanup add journal: %w", err), ErrorScaffoldFailed)
	}

	if !req.Verify {
		return nil
	}

	if err := s.verify(ctx, req.Dir, runtime, verificationPlan{Timeout: req.VerifyTimeout, Tidy: true}, &output.Checks); err != nil {
		output.VerifyPassed = false
		return &OperationError{Kind: ErrorVerifyFailed, Partial: true, Err: fmt.Errorf("verification failed: %w", err)}
	}
	output.VerifyPassed = true

	return nil
}

// applyAddFileWrites writes every rendered integration file under a recovery
// journal so a later failure (dependency install, manifest update) can revert
// the whole set. It mirrors applyPlanWrites in scaffolder_apply.go: each file is
// journaled BEFORE it is written, and a write failure rolls back what landed so
// far. rollbackActionForPath distinguishes create (file absent -> remove on
// rollback) from overwrite (file present -> restore original content+mode).
func applyAddFileWrites(req addRequest, plan *addPlan) (*applyJournal, []rollbackAction, error) {
	journal, err := beginApplyJournal(req.Dir)
	if err != nil {
		return nil, nil, wrapAddFlowError(fmt.Errorf("begin add journal: %w", err), ErrorScaffoldFailed)
	}

	applied := make([]rollbackAction, 0, len(plan.RenderedPaths))
	for _, relPath := range plan.RenderedPaths {
		destPath := filepath.Join(req.Dir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			rbErr := rollbackWithError(fmt.Errorf("create directory for %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, ErrorScaffoldFailed)
		}

		rendered := plan.RenderedFiles[relPath]
		action, err := rollbackActionForPath(req.Dir, destPath)
		if err != nil {
			rbErr := rollbackWithError(fmt.Errorf("prepare rollback for %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, ErrorScaffoldFailed)
		}
		action.ExpectedExists = true
		action.ExpectedHash = sha256Hex(rendered)
		action.ExpectedSize = int64(len(rendered))
		action.ExpectedMode = 0644

		idx, err := journal.appendAction(action)
		if err != nil {
			rbErr := rollbackWithError(fmt.Errorf("record rollback intent for %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, ErrorScaffoldFailed)
		}

		if err := writeFileAtomic(destPath, rendered, 0644); err != nil {
			rbErr := rollbackWithError(fmt.Errorf("write %s: %w", relPath, err), applied, journal)
			return nil, nil, wrapAddFlowError(rbErr, ErrorScaffoldFailed)
		}

		if err := journal.markActionApplied(idx); err != nil {
			rbErr := rollbackWithError(fmt.Errorf("mark rollback intent applied for %s: %w", relPath, err), append(applied, action), journal)
			return nil, nil, wrapAddFlowError(rbErr, ErrorScaffoldFailed)
		}
		action.Mode = 0644
		applied = append(applied, action)
	}

	return journal, applied, nil
}
