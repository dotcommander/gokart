package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func hasLegacyGreetAppField(path string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return false
	}
	for _, decl := range file.Decls {
		if legacyGreetDeclaration(decl) {
			return true
		}
	}
	return false
}

func legacyGreetDeclaration(decl ast.Decl) bool {
	gen, ok := decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.TYPE {
		return false
	}
	for _, spec := range gen.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "GreetCommand" {
			continue
		}
		fields, ok := typeSpec.Type.(*ast.StructType)
		return ok && hasLegacyAppField(fields.Fields.List)
	}
	return false
}

func hasLegacyAppField(fields []*ast.Field) bool {
	for _, field := range fields {
		if isLegacyAppField(field) {
			return true
		}
	}
	return false
}

func isLegacyAppField(field *ast.Field) bool {
	if len(field.Names) != 1 || field.Names[0].Name != "App" {
		return false
	}
	star, ok := field.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := star.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Context" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "app"
}

func renderIntegrationFiles(data TemplateData) (map[string][]byte, error) {
	// Add only operates on managed structured projects. Refresh derived template
	// policy after callers merge the requested integration booleans.
	data.derive(true)
	result := make(map[string][]byte)
	templatePaths := []struct {
		tmpl    string
		outPath string
	}{
		{templateRootStructured + "/internal/app/context.go.tmpl", appContextPath},
		{templateRootStructured + "/internal/commands/root.go.tmpl", commandsRootPath},
	}

	for _, tp := range templatePaths {
		rendered, err := renderTemplate(templates, tp.tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", tp.outPath, err)
		}
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
		return fileSafetyConflict
	}

	currentHash := sha256Hex(existing)
	for _, f := range manifest.Files {
		if f.Path == relPath {
			if f.ContentSHA256 == currentHash {
				return fileSafetySafe
			}
			return fileSafetyConflict
		}
	}
	return fileSafetyConflict
}

func (s *Service) addGoDependencies(ctx context.Context, dir string, integrations []string, runtime Runtime, checks *[]CheckResult) error {
	data := TemplateData{}
	for _, name := range integrations {
		entry, ok := integrationRegistry[name]
		if ok {
			entry.setTemplate(&data)
		}
	}
	packages := integrationDependencies(data)
	if len(packages) == 0 {
		return nil
	}

	goGetArgs := append([]string{"get"}, packages...)
	if err := s.runCheckedGoCommand(ctx, dir, runtime, checks, goGetArgs...); err != nil {
		return fmt.Errorf("go get: %w", err)
	}
	if err := s.runCheckedGoCommand(ctx, dir, runtime, checks, "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}
	return nil
}

func updateAddManifest(dir string, manifest *scaffoldManifest, data TemplateData, renderedFiles map[string][]byte) error {
	manifest.Version = scaffoldManifestV2
	manifest.Integrations = &manifestIntegrations{
		SQLite: data.UseSQLite, Postgres: data.UsePostgres, AI: data.UseAI, Redis: data.UseRedis,
	}
	if manifest.Module == "" {
		manifest.Module = data.Module
	}
	if manifest.Mode == "" {
		manifest.Mode = modeStructured
	}
	if manifest.UseGlobal == nil {
		manifest.UseGlobal = boolPtr(data.UseGlobal)
	}

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
				Path: relPath, Action: manifestActionCreate, TemplateSHA256: hash, ContentSHA256: hash, Mode: 0644,
			})
		}
	}

	slices.SortFunc(manifest.Files, func(a, b scaffoldManifestFile) int {
		return strings.Compare(a.Path, b.Path)
	})
	now := time.Now().UTC()
	manifest.GeneratedAt = &now
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')
	return writeScaffoldManifest(dir, manifestData)
}
