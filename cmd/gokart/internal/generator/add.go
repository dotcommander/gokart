package generator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	integrationSQLite   = "sqlite"
	integrationPostgres = "postgres"
	integrationAI       = "ai"
	integrationRedis    = "redis"
)

// integrationEntry is the single source of truth for one integration: its
// go.mod packages plus how to read/write its bit on a manifestIntegrations.
// Adding a new integration means adding one entry here — nothing else.
type integrationEntry struct {
	deps        []string
	template    func(TemplateData) bool
	setTemplate func(*TemplateData)
	get         func(*manifestIntegrations) bool
	set         func(*manifestIntegrations, bool)
	description string
	flag        string
	golden      string
	environment string
	upstream    string
	recipe      string
}

// integrationRegistry is the ONLY place that enumerates integration names.
var integrationRegistry = map[string]integrationEntry{
	integrationSQLite: {
		deps:        []string{"github.com/dotcommander/gokart/sqlite@" + defaultGokartSQLiteVersion},
		template:    func(data TemplateData) bool { return data.UseSQLite },
		setTemplate: func(data *TemplateData) { data.UseSQLite = true },
		get:         func(m *manifestIntegrations) bool { return m.SQLite },
		set:         func(m *manifestIntegrations, v bool) { m.SQLite = v },
		description: "SQLite through sqlite.Open",
		flag:        "--db sqlite", golden: "structured-sqlite",
		environment: "{{APP}}_DB_PATH (optional; defaults to the user cache directory)",
		upstream:    "modernc.org/sqlite",
		recipe:      "SQLite transaction",
	},
	integrationPostgres: {
		deps:        []string{"github.com/dotcommander/gokart/postgres@" + defaultGokartPostgresVersion, "github.com/jackc/pgx/v5@" + defaultPGXVersion},
		template:    func(data TemplateData) bool { return data.UsePostgres },
		setTemplate: func(data *TemplateData) { data.UsePostgres = true },
		get:         func(m *manifestIntegrations) bool { return m.Postgres },
		set:         func(m *manifestIntegrations, v bool) { m.Postgres = v },
		description: "PostgreSQL through postgres.Open",
		flag:        "--db postgres", golden: "structured-postgres",
		environment: "DATABASE_URL (required by PostgreSQL commands)",
		upstream:    "pgxpool.NewWithConfig",
		recipe:      "PostgreSQL transaction",
	},
	integrationAI: {
		deps:        []string{"github.com/openai/openai-go/v3@" + defaultOpenAIVersion},
		template:    func(data TemplateData) bool { return data.UseAI },
		setTemplate: func(data *TemplateData) { data.UseAI = true },
		get:         func(m *manifestIntegrations) bool { return m.AI },
		set:         func(m *manifestIntegrations, v bool) { m.AI = v },
		description: "OpenAI through the official SDK",
		flag:        "--ai", golden: "structured-ai",
		environment: "OPENAI_API_KEY (required by AI commands)",
		upstream:    "openai.NewClient",
		recipe:      "Add integrations",
	},
	integrationRedis: {
		deps:        []string{"github.com/dotcommander/gokart/cache@" + defaultGokartCacheVersion, "github.com/redis/go-redis/v9@" + defaultRedisVersion},
		template:    func(data TemplateData) bool { return data.UseRedis },
		setTemplate: func(data *TemplateData) { data.UseRedis = true },
		get:         func(m *manifestIntegrations) bool { return m.Redis },
		set:         func(m *manifestIntegrations, v bool) { m.Redis = v },
		description: "Redis through cache.Open",
		flag:        "--redis", golden: "structured-redis",
		environment: "REDIS_ADDR (optional; defaults to localhost:6379)",
		upstream:    "redis.NewClient",
		recipe:      "Redis with direct client access",
	},
}

type integrationReadmeData struct {
	Name        string
	Description string
	Environment string
}

func selectedIntegrationReadmeData(data TemplateData) []integrationReadmeData {
	names := make([]string, 0, len(integrationRegistry))
	for name, entry := range integrationRegistry {
		if entry.template(data) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	selected := make([]integrationReadmeData, 0, len(names))
	for _, name := range names {
		entry := integrationRegistry[name]
		environment := strings.ReplaceAll(entry.environment, "{{APP}}", strings.ToUpper(data.Name))
		selected = append(selected, integrationReadmeData{Name: name, Description: entry.description, Environment: environment})
	}
	return selected
}

func integrationDependencies(data TemplateData) []string {
	var packages []string
	for _, entry := range integrationRegistry {
		if entry.template(data) {
			packages = append(packages, entry.deps...)
		}
	}
	sort.Strings(packages)
	return packages
}

type addCommandOutput = AddResult

type fileSafetyResult int

const (
	fileSafetyCreate   fileSafetyResult = iota // file doesn't exist → create
	fileSafetySafe                             // hash matches manifest → safe overwrite
	fileSafetyConflict                         // hash differs → conflict
)

func isFlatProject(manifest *scaffoldManifest) bool {
	return manifest.TemplateRoot == templateRootFlat || manifest.Mode == modeFlat
}

func readAddManifest(dir string) (*scaffoldManifest, error) {
	path := filepath.Join(dir, scaffoldManifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no manifest found at %s (run gokart new to create a project first)", path)
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest scaffoldManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

func detectCurrentIntegrations(manifest *scaffoldManifest, dir string) (string, *manifestIntegrations) {
	// v2 manifests have integrations directly
	if manifest.Integrations != nil {
		return manifest.Module, manifest.Integrations
	}

	// v1: infer from go.mod
	return inferIntegrationsFromGoMod(dir)
}

func inferIntegrationsFromGoMod(dir string) (string, *manifestIntegrations) {
	result := &manifestIntegrations{}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", result
	}
	content := string(data)

	var module string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			module = strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			break
		}
	}

	if strings.Contains(content, "gokart/sqlite") {
		result.SQLite = true
	}
	if strings.Contains(content, "gokart/postgres") {
		result.Postgres = true
	}
	if strings.Contains(content, "openai/openai-go/v3") {
		result.AI = true
	}
	if strings.Contains(content, "gokart/cache") {
		result.Redis = true
	}
	return module, result
}

func integrationEnabled(current *manifestIntegrations, name string) bool {
	if current == nil {
		return false
	}
	entry, ok := integrationRegistry[name]
	if !ok {
		return false
	}
	return entry.get(current)
}

func setIntegration(m *manifestIntegrations, name string, enable bool) {
	if m == nil {
		return
	}
	if entry, ok := integrationRegistry[name]; ok {
		entry.set(m, enable)
	}
}

func mergeIntegrations(current *manifestIntegrations, toAdd []string) *manifestIntegrations {
	merged := &manifestIntegrations{}
	if current != nil {
		*merged = *current
	}
	for _, name := range toAdd {
		setIntegration(merged, name, true)
	}
	return merged
}

func inferTemplateData(manifest *scaffoldManifest, dir string, current *manifestIntegrations, toAdd []string, goModModule string) (TemplateData, error) {
	// Get module from manifest or go.mod
	module := manifest.Module
	if module == "" {
		module = goModModule
		if module == "" {
			return TemplateData{}, errors.New("no module found in manifest or go.mod")
		}
	}

	name := filepath.Base(module)

	// Determine UseGlobal
	useGlobal := false
	if manifest.UseGlobal != nil {
		useGlobal = *manifest.UseGlobal
	} else {
		// Infer: does internal/app/config.go exist?
		if _, err := os.Stat(filepath.Join(dir, "internal", "app", "config.go")); err == nil {
			useGlobal = true
		}
	}

	// Determine IncludeExample
	includeExample := false
	if _, err := os.Stat(filepath.Join(dir, "internal", "commands", "greet.go")); err == nil {
		includeExample = true
	}

	// Merge current + requested integrations
	merged := mergeIntegrations(current, toAdd)

	data := baseTemplateData(name, module, useGlobal, includeExample)
	data.UseSQLite = merged.SQLite
	data.UsePostgres = merged.Postgres
	data.UseAI = merged.AI
	data.UseRedis = merged.Redis
	data.Integrations = selectedIntegrationReadmeData(data)
	data.LegacyGreetAppField = hasLegacyGreetAppField(filepath.Join(dir, "internal", "commands", "greet.go"))

	return data, nil
}
