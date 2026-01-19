// Example: Configuration loading with GoKart config.
//
// This example demonstrates:
//   - Loading YAML configuration files
//   - Environment variable binding with automatic key mapping
//   - Type-safe configuration with generics
//   - Default values
//
// To run this example, first create config.yaml:
//
//	port: 8080
//	database:
//	  host: localhost
//	  port: 5432
//	  name: mydb
//	features:
//	  enable_cache: true
//	  cache_ttl: 3600
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dotcommander/gokart"
)

// Config represents your application configuration.
// The mapstructure tags bind YAML/JSON keys to struct fields.
type Config struct {
	Port int `mapstructure:"port"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`

	Features struct {
		EnableCache bool   `mapstructure:"enable_cache"`
		CacheTTL    int    `mapstructure:"cache_ttl"`
		DebugMode   bool   `mapstructure:"debug_mode"`
	} `mapstructure:"features"`
}

func main() {
	// Example 1: Load config with default values
	defaults := Config{
		Port: 3000, // Default port if not in config file
		Database: struct {
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Name     string `mapstructure:"name"`
			Password string `mapstructure:"password"`
		}{
			Host: "localhost",
			Port: 5432,
		},
	}

	cfg, err := gokart.LoadConfigWithDefaults(defaults, "config.yaml")
	if err != nil {
		log.Printf("Warning: %v (using defaults)", err)
		cfg = defaults
	}

	fmt.Printf("Loaded config:\n")
	fmt.Printf("  Port: %d\n", cfg.Port)
	fmt.Printf("  DB Host: %s\n", cfg.Database.Host)
	fmt.Printf("  DB Name: %s\n", cfg.Database.Name)
	fmt.Printf("  Cache enabled: %v\n", cfg.Features.EnableCache)

	// Example 2: Environment variable override
	// Set env vars to override config values:
	//   export PORT=9000
	//   export DATABASE_HOST=prod-db.example.com
	//   export DATABASE_NAME=production
	//   export FEATURES_ENABLE_CACHE=false
	//
	// GoKart automatically maps:
	//   - db.host -> DATABASE_HOST
	//   - database.name -> DATABASE_NAME
	//   - features.enable_cache -> FEATURES_ENABLE_CACHE

	fmt.Println("\nEnvironment overrides (if set):")
	fmt.Printf("  PORT env: %s\n", os.Getenv("PORT"))
	fmt.Printf("  DATABASE_HOST env: %s\n", os.Getenv("DATABASE_HOST"))

	// Example 3: Multiple config paths (first found wins)
	// Useful for supporting both YAML and JSON:
	//
	// cfg, err := gokart.LoadConfig[Config](
	//     "config.yaml",
	//     "config.json",
	//     "config.yml",
	// )

	// Example 4: Access nested config
	fmt.Println("\nNested access:")
	fmt.Printf("  Cache TTL: %d seconds\n", cfg.Features.CacheTTL)

	// Example 5: Validation pattern
	if err := validateConfig(&cfg); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	fmt.Println("\nConfig is valid!")
}

// validateConfig demonstrates configuration validation
func validateConfig(cfg *Config) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}
