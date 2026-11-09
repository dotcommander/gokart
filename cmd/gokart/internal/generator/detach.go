package generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	gokartImportPrefix = "github.com/dotcommander/gokart"
	appConfigPath      = "internal/app/config.go"
)

type detachPlan struct {
	Manifest      *scaffoldManifest
	RenderedFiles map[string][]byte
	RenderedPaths []string
	GoModCoupled  bool
}

// Detach rewrites generated GoKart wiring files to import upstream packages
// directly, tidies the module, and removes the manifest.
func (s *Service) Detach(ctx context.Context, request DetachRequest, runtime Runtime) (DetachResult, error) {
	result := DetachResult{DryRun: request.DryRun, VerifyRequested: request.Verify}
	var err error
	mutate := func() error {
		plan, err := planDetach(request, &result)
		if err != nil {
			return err
		}
		if plan == nil || request.DryRun {
			return nil
		}
		return s.applyDetach(ctx, request, plan, &result, runtime)
	}
	if request.DryRun {
		err = mutate()
	} else {
		err = withTargetMutationLock(request.Dir, mutate)
	}
	return result, err
}

func planDetach(req DetachRequest, result *DetachResult) (*detachPlan, error) {
	manifest, done, err := detachManifest(req, result)
	if err != nil || done {
		return nil, err
	}
	goMod, err := os.ReadFile(filepath.Join(req.Dir, "go.mod"))
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("read go.mod: %w", err), ErrorScaffoldFailed)
	}
	plan := &detachPlan{Manifest: manifest, GoModCoupled: bytes.Contains(goMod, []byte(gokartImportPrefix))}
	rendered, err := renderDetachData(manifest, req.Dir)
	if err != nil {
		return nil, err
	}
	if err := collectDetachFiles(req, manifest, rendered, plan, result); err != nil {
		return nil, err
	}
	sort.Strings(plan.RenderedPaths)
	result.FilesOverwritten = append(result.FilesOverwritten, plan.RenderedPaths...)
	if len(plan.RenderedPaths) == 0 && !plan.GoModCoupled {
		result.AlreadyDetached = true
		return nil, nil
	}
	return plan, nil
}

// detachManifest returns the manifest, or done=true when the request needs
// no further planning (already-detached marker is set on result).
func detachManifest(req DetachRequest, result *DetachResult) (*scaffoldManifest, bool, error) {
	if _, err := os.Stat(filepath.Join(req.Dir, scaffoldManifestPath)); os.IsNotExist(err) && looksLikeFlatProject(req.Dir) {
		return nil, false, wrapAddFlowError(errors.New("gokart detach: flat projects have no GoKart runtime import; nothing to detach"), ErrorFlatModeUnsupported)
	}
	if _, statErr := os.Stat(filepath.Join(req.Dir, scaffoldManifestPath)); os.IsNotExist(statErr) && projectFreeOfGokart(req.Dir) {
		result.AlreadyDetached = true
		return nil, true, nil
	}
	manifest, err := readAddManifest(req.Dir)
	if err != nil {
		return nil, false, wrapAddFlowError(err, ErrorManifestNotFound)
	}
	if isFlatProject(manifest) {
		return nil, false, wrapAddFlowError(errors.New("gokart detach: flat projects have no GoKart runtime import; nothing to detach"), ErrorFlatModeUnsupported)
	}
	return manifest, false, nil
}

func renderDetachData(manifest *scaffoldManifest, dir string) (map[string][]byte, error) {
	_, current := detectCurrentIntegrations(manifest, dir)
	data, err := inferTemplateData(manifest, dir, current, nil, manifest.Module)
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("infer project state: %w", err), ErrorScaffoldFailed)
	}
	rendered, err := renderDetachFiles(data)
	if err != nil {
		return nil, wrapAddFlowError(fmt.Errorf("render templates: %w", err), ErrorScaffoldFailed)
	}
	return rendered, nil
}

func collectDetachFiles(req DetachRequest, manifest *scaffoldManifest, rendered map[string][]byte, plan *detachPlan, result *DetachResult) error {
	plan.RenderedFiles = make(map[string][]byte)
	for relPath, content := range rendered {
		existing, err := os.ReadFile(filepath.Join(req.Dir, relPath))
		if err != nil || !bytes.Contains(existing, []byte(gokartImportPrefix)) {
			continue // missing or already free of GoKart imports
		}
		if checkFileSafety(req.Dir, relPath, manifest) == fileSafetyConflict {
			if !req.Force {
				return &OperationError{Kind: ErrorExistingFileConflict, Conflicts: []string{relPath}, Err: fmt.Errorf("file %s has been modified (use --force to overwrite)", relPath)}
			}
			result.Warnings = append(result.Warnings, "force-overwriting modified file: "+relPath)
		}
		plan.RenderedFiles[relPath] = content
		plan.RenderedPaths = append(plan.RenderedPaths, relPath)
	}
	return nil
}

// projectFreeOfGokart reports whether go.mod and the generated wiring files
// contain no GoKart references. A missing go.mod means it is not a project.
func projectFreeOfGokart(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	for _, rel := range []string{"go.mod", appContextPath, appConfigPath, "internal/commands/root.go"} {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false
		}
		if bytes.Contains(data, []byte(gokartImportPrefix)) {
			return false
		}
	}
	return true
}

func renderDetachFiles(data TemplateData) (map[string][]byte, error) {
	files, err := renderIntegrationFiles(data)
	if err != nil {
		return nil, err
	}
	if data.UseGlobal {
		rendered, err := renderTemplate(templates, templateRootStructured+"/internal/app/config.go.tmpl", data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", appConfigPath, err)
		}
		if len(bytes.TrimSpace(rendered)) > 0 {
			files[appConfigPath] = rendered
		}
	}
	return files, nil
}

func (s *Service) applyDetach(ctx context.Context, req DetachRequest, plan *detachPlan, result *DetachResult, runtime Runtime) error {
	addReq := addRequest{Dir: req.Dir}
	journal, applied, err := applyAddFileWrites(addReq, &addPlan{RenderedFiles: plan.RenderedFiles, RenderedPaths: plan.RenderedPaths})
	if err != nil {
		return err
	}
	depActions, err := journalDependencyFiles(req.Dir, journal)
	if err != nil {
		return wrapAddFlowError(rollbackWithError(fmt.Errorf("prepare dependency rollback: %w", err), applied, journal), ErrorScaffoldFailed)
	}
	tidyErr := s.runCheckedGoCommand(ctx, req.Dir, runtime, &result.Checks, "mod", "tidy")
	applied, err = markDependencyFilesApplied(depActions, journal, applied)
	if tidyErr != nil {
		if err != nil {
			tidyErr = errors.Join(tidyErr, fmt.Errorf("record dependency rollback state: %w", err))
		}
		return wrapAddFlowError(rollbackWithError(fmt.Errorf("go mod tidy: %w", tidyErr), applied, journal), ErrorScaffoldFailed)
	}
	if err != nil {
		return wrapAddFlowError(rollbackWithError(fmt.Errorf("record dependency rollback state: %w", err), applied, journal), ErrorScaffoldFailed)
	}
	if err := journal.markCompleted(); err != nil {
		return wrapAddFlowError(fmt.Errorf("finalize detach journal: %w", err), ErrorScaffoldFailed)
	}
	if err := journal.cleanup(); err != nil {
		return wrapAddFlowError(fmt.Errorf("cleanup detach journal: %w", err), ErrorScaffoldFailed)
	}
	if err := os.Remove(filepath.Join(req.Dir, scaffoldManifestPath)); err != nil && !os.IsNotExist(err) {
		return wrapAddFlowError(fmt.Errorf("remove manifest: %w", err), ErrorScaffoldFailed)
	}
	result.ManifestRemoved = true
	if !req.Verify {
		return nil
	}
	if err := s.verify(ctx, req.Dir, runtime, verificationPlan{Timeout: req.VerifyTimeout}, &result.Checks); err != nil {
		return &OperationError{Kind: ErrorVerifyFailed, Partial: true, Err: fmt.Errorf("verification failed: %w", err)}
	}
	result.VerifyPassed = true
	return nil
}
