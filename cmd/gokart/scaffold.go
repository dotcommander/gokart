package main

import (
	"embed"
	"runtime"
	"strings"
)

//go:embed all:templates
var templates embed.FS

const (
	defaultGokartCLIVersion      = "v0.0.0-20260301050059-af15bf731eeb"
	defaultGokartSQLiteVersion   = "v0.0.0-20260301050059-af15bf731eeb"
	defaultGokartPostgresVersion = "v0.0.0-20260301050059-af15bf731eeb"
	defaultGokartAIVersion       = "v0.0.0-20260301050059-af15bf731eeb"

	defaultCobraVersion  = "v1.10.2"
	defaultViperVersion  = "v1.21.0"
	defaultPGXVersion    = "v5.8.0"
	defaultOpenAIVersion = "v3.24.0"
	defaultGooseVersion  = "v3.27.0"
)

// TemplateData holds variables for template substitution.
type TemplateData struct {
	Name           string
	Module         string
	GoVersion      string
	UseSQLite      bool
	UsePostgres    bool
	UseAI          bool
	IncludeExample bool
	UseGlobal      bool

	GokartCLIVersion      string
	GokartSQLiteVersion   string
	GokartPostgresVersion string
	GokartAIVersion       string
	CobraVersion          string
	ViperVersion          string
	PGXVersion            string
	OpenAIVersion         string
	GooseVersion          string
}

// ScaffoldFlat creates a flat project structure with a single main.go.
func ScaffoldFlat(dir, name, module string, useGlobal, includeExample bool, opts ApplyOptions) (*ApplyResult, error) {
	data := baseTemplateData(name, module, useGlobal, includeExample)
	return Apply(templates, "templates/flat", dir, data, opts)
}

// ScaffoldStructured creates a structured project with cmd/, internal/commands/, internal/actions/.
func ScaffoldStructured(dir, name, module string, useSQLite, usePostgres, useAI, useGlobal, includeExample bool, opts ApplyOptions) (*ApplyResult, error) {
	data := baseTemplateData(name, module, useGlobal, includeExample)
	data.UseSQLite = useSQLite
	data.UsePostgres = usePostgres
	data.UseAI = useAI
	return Apply(templates, "templates/structured", dir, data, opts)
}

func baseTemplateData(name, module string, useGlobal, includeExample bool) TemplateData {
	return TemplateData{
		Name:           name,
		Module:         module,
		GoVersion:      goVersion(),
		IncludeExample: includeExample,
		UseGlobal:      useGlobal,

		GokartCLIVersion:      defaultGokartCLIVersion,
		GokartSQLiteVersion:   defaultGokartSQLiteVersion,
		GokartPostgresVersion: defaultGokartPostgresVersion,
		GokartAIVersion:       defaultGokartAIVersion,
		CobraVersion:          defaultCobraVersion,
		ViperVersion:          defaultViperVersion,
		PGXVersion:            defaultPGXVersion,
		OpenAIVersion:         defaultOpenAIVersion,
		GooseVersion:          defaultGooseVersion,
	}
}

// goVersion returns the current Go version without the "go" prefix.
func goVersion() string {
	v := runtime.Version()
	// runtime.Version() returns "go1.24.0", we want "1.24.0"
	return strings.TrimPrefix(v, "go")
}
