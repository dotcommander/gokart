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
	defaultGokartCacheVersion    = "v0.0.0-20260301050059-af15bf731eeb"

	defaultCobraVersion  = "v1.10.2"
	defaultViperVersion  = "v1.21.0"
	defaultPGXVersion    = "v5.8.0"
	defaultOpenAIVersion = "v3.24.0"
	defaultRedisVersion  = "v9.7.0"
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
	UseRedis       bool
	IncludeExample bool
	UseGlobal      bool

	GokartCLIVersion      string
	GokartSQLiteVersion   string
	GokartPostgresVersion string
	GokartAIVersion       string
	GokartCacheVersion    string
	RedisVersion          string
	CobraVersion          string
	ViperVersion          string
	PGXVersion            string
	OpenAIVersion         string
	GooseVersion          string
}

type scaffoldSpec struct {
	Dir          string
	TemplateRoot string
	Data         TemplateData
	Metadata     scaffoldManifestMetadata
}

// ScaffoldFlat creates a flat project structure with a single main.go.
//
//nolint:revive // public API, params are distinct
func ScaffoldFlat(dir, name, module string, useGlobal, includeExample bool, opts ApplyOptions) (*ApplyResult, error) {
	return applyScaffoldSpec(scaffoldSpec{
		Dir:          dir,
		TemplateRoot: "templates/flat",
		Data:         baseTemplateData(name, module, useGlobal, includeExample),
		Metadata: scaffoldManifestMetadata{
			Mode:      modeFlat,
			Module:    module,
			UseGlobal: boolPtr(useGlobal),
		},
	}, opts)
}

// ScaffoldStructured creates a structured project with cmd/, internal/commands/, internal/actions/.
//
//nolint:revive // public API, boolean flags for each integration
func ScaffoldStructured(dir, name, module string, useSQLite, usePostgres, useAI, useRedis, useGlobal, includeExample bool, opts ApplyOptions) (*ApplyResult, error) {
	data := baseTemplateData(name, module, useGlobal, includeExample)
	data.UseSQLite = useSQLite
	data.UsePostgres = usePostgres
	data.UseAI = useAI
	data.UseRedis = useRedis

	return applyScaffoldSpec(scaffoldSpec{
		Dir:          dir,
		TemplateRoot: "templates/structured",
		Data:         data,
		Metadata: scaffoldManifestMetadata{
			Integrations: &manifestIntegrations{
				SQLite:   useSQLite,
				Postgres: usePostgres,
				AI:       useAI,
				Redis:    useRedis,
			},
			Mode:      modeStructured,
			Module:    module,
			UseGlobal: boolPtr(useGlobal),
		},
	}, opts)
}

func applyScaffoldSpec(spec scaffoldSpec, opts ApplyOptions) (*ApplyResult, error) {
	opts.ManifestMetadata = &spec.Metadata
	return Apply(templates, spec.TemplateRoot, spec.Dir, spec.Data, opts)
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
		GokartCacheVersion:    defaultGokartCacheVersion,
		RedisVersion:          defaultRedisVersion,
		CobraVersion:          defaultCobraVersion,
		ViperVersion:          defaultViperVersion,
		PGXVersion:            defaultPGXVersion,
		OpenAIVersion:         defaultOpenAIVersion,
		GooseVersion:          defaultGooseVersion,
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// goVersion returns the current Go version without the "go" prefix.
func goVersion() string {
	v := runtime.Version()
	// runtime.Version() returns "go1.24.0", we want "1.24.0"
	return strings.TrimPrefix(v, "go")
}
