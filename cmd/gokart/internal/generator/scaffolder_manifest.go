package generator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

func renderScaffoldManifest(templateRoot string, plan []plannedFile, opts ApplyOptions) ([]byte, error) {
	manifest, err := buildScaffoldManifest(templateRoot, plan, opts)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode scaffold manifest: %w", err)
	}
	return append(data, '\n'), nil
}

func writeScaffoldManifest(targetRoot string, manifestData []byte) error {
	manifestPath, err := safeDestinationPath(targetRoot, scaffoldManifestPath)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("write manifest file %s: %w", filepath.ToSlash(manifestPath), err)
	}
	return nil
}

func buildScaffoldManifest(templateRoot string, plan []plannedFile, opts ApplyOptions) (scaffoldManifest, error) {
	files := make([]scaffoldManifestFile, 0, len(plan))
	for _, file := range plan {
		actionLabel, err := planActionLabel(file.Action)
		if err != nil {
			return scaffoldManifest{}, fmt.Errorf("build manifest for %s: %w", file.RelPath, err)
		}
		content := manifestContentForPlannedFile(file)
		mode := manifestModeForPlannedFile(file)
		files = append(files, scaffoldManifestFile{
			Path: file.RelPath, Action: actionLabel, TemplateSHA256: sha256Hex(file.Rendered),
			ContentSHA256: sha256Hex(content), Mode: uint32(mode),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	meta := opts.ManifestMetadata
	version := scaffoldManifestV1
	if meta != nil {
		version = scaffoldManifestV2
	}
	m := scaffoldManifest{
		Version: version, Generator: manifestGenerator, GeneratorVersion: opts.GeneratorVersion,
		TemplateRoot: templateRoot, ExistingFilePolicy: opts.ExistingFilePolicy, Files: files,
	}
	if meta != nil {
		m.Integrations = meta.Integrations
		m.Mode = meta.Mode
		m.Module = meta.Module
		m.UseGlobal = meta.UseGlobal
	}
	return m, nil
}

func planActionLabel(action planAction) (string, error) {
	switch action {
	case planActionCreate:
		return manifestActionCreate, nil
	case planActionOverwrite:
		return "overwrite", nil
	case planActionSkip:
		return "skip", nil
	case planActionUnchanged:
		return "unchanged", nil
	default:
		return "", fmt.Errorf("unknown plan action %d", action)
	}
}

func manifestContentForPlannedFile(file plannedFile) []byte {
	switch file.Action {
	case planActionCreate, planActionOverwrite:
		return file.Rendered
	case planActionSkip, planActionUnchanged:
		return file.Existing
	default:
		return file.Rendered
	}
}

func manifestModeForPlannedFile(file plannedFile) fs.FileMode {
	switch file.Action {
	case planActionCreate:
		return 0644
	case planActionOverwrite, planActionSkip, planActionUnchanged:
		if file.ExistingMode != 0 {
			return file.ExistingMode
		}
		return 0644
	default:
		return 0644
	}
}
