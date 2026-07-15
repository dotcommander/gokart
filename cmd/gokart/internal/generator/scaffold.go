package generator

import (
	"embed"
)

//go:embed all:templates
var templates embed.FS

const (
	defaultGokartVersion         = "v0.13.0"
	defaultGokartSQLiteVersion   = defaultGokartVersion
	defaultGokartPostgresVersion = defaultGokartVersion
	defaultGokartCacheVersion    = defaultGokartVersion

	defaultKongVersion   = "v1.15.0"
	defaultViperVersion  = "v1.21.0"
	defaultPGXVersion    = "v5.10.0"
	defaultOpenAIVersion = "v3.42.0"
	defaultRedisVersion  = "v9.21.0"
	defaultGooseVersion  = "v3.27.2"
	generatedGoVersion   = "1.26.0"
)

// TemplateData holds variables for template substitution.
type TemplateData struct {
	Name                string
	Module              string
	GoVersion           string
	UseSQLite           bool
	UsePostgres         bool
	UseAI               bool
	UseRedis            bool
	IncludeExample      bool
	UseGlobal           bool
	HasIntegrations     bool
	HasAppPackage       bool
	Managed             bool
	LegacyGreetAppField bool
	Integrations        []integrationReadmeData

	GokartSQLiteVersion   string
	GokartPostgresVersion string
	GokartCacheVersion    string
	RedisVersion          string
	KongVersion           string
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
	opts.SkipManifest = true
	data := baseTemplateData(name, module, useGlobal, includeExample)
	data.derive(false)
	return applyScaffoldSpec(scaffoldSpec{
		Dir:          dir,
		TemplateRoot: templateRootFlat,
		Data:         data,
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
	data.derive(!opts.SkipManifest)
	data.Integrations = selectedIntegrationReadmeData(data)

	return applyScaffoldSpec(scaffoldSpec{
		Dir:          dir,
		TemplateRoot: templateRootStructured,
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
		GoVersion:      generatedGoVersion,
		IncludeExample: includeExample,
		UseGlobal:      useGlobal,

		GokartSQLiteVersion:   defaultGokartSQLiteVersion,
		GokartPostgresVersion: defaultGokartPostgresVersion,
		GokartCacheVersion:    defaultGokartCacheVersion,
		RedisVersion:          defaultRedisVersion,
		KongVersion:           defaultKongVersion,
		ViperVersion:          defaultViperVersion,
		PGXVersion:            defaultPGXVersion,
		OpenAIVersion:         defaultOpenAIVersion,
		GooseVersion:          defaultGooseVersion,
	}
}

func (d *TemplateData) derive(managed bool) {
	d.HasIntegrations = d.UseSQLite || d.UsePostgres || d.UseAI || d.UseRedis
	d.HasAppPackage = d.UseGlobal || d.HasIntegrations
	d.Managed = managed
}

func boolPtr(b bool) *bool {
	return &b
}
