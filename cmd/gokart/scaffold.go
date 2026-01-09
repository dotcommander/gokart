package main

import (
	"embed"
	"runtime"
	"strings"
)

//go:embed templates
var templates embed.FS

// TemplateData holds variables for template substitution.
type TemplateData struct {
	Name        string
	Module      string
	GoVersion   string
	UseSQLite   bool
	UsePostgres bool
	UseAI       bool
	UseGlobal   bool
}

// ScaffoldFlat creates a flat project structure with a single main.go.
func ScaffoldFlat(dir, name, module string, useGlobal bool) error {
	data := TemplateData{
		Name:      name,
		Module:    module,
		GoVersion: goVersion(),
		UseGlobal: useGlobal,
	}
	return Apply(templates, "templates/flat", dir, data)
}

// ScaffoldStructured creates a structured project with cmd/, internal/commands/, internal/actions/.
func ScaffoldStructured(dir, name, module string, useSQLite, usePostgres, useAI, useGlobal bool) error {
	data := TemplateData{
		Name:        name,
		Module:      module,
		GoVersion:   goVersion(),
		UseSQLite:   useSQLite,
		UsePostgres: usePostgres,
		UseAI:       useAI,
		UseGlobal:   useGlobal,
	}
	return Apply(templates, "templates/structured", dir, data)
}

// goVersion returns the current Go version without the "go" prefix.
func goVersion() string {
	v := runtime.Version()
	// runtime.Version() returns "go1.24.0", we want "1.24.0"
	return strings.TrimPrefix(v, "go")
}
