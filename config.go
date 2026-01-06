package gokart

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from the first available file path into type T.
//
// Features:
//   - Supports multiple config paths (first found wins)
//   - Automatic environment variable binding
//   - DOT to UNDERSCORE env key mapping (e.g., db.host → DB_HOST)
//
// Supported formats: JSON, YAML, TOML, HCL, envfile, Java properties
//
// Example:
//
//	type Config struct {
//	    DB struct {
//	        Host string `mapstructure:"host"`
//	        Port int    `mapstructure:"port"`
//	    } `mapstructure:"db"`
//	}
//	cfg, err := gokart.LoadConfig[Config]("config.yaml", "config.json")
func LoadConfig[T any](paths ...string) (T, error) {
	var zero T
	return LoadConfigWithDefaults(zero, paths...)
}

// LoadConfigWithDefaults loads configuration with default values pre-populated.
//
// The defaults parameter provides fallback values that will be overridden
// by values from config files or environment variables.
//
// Example:
//
//	defaults := Config{
//	    DB: struct{Host string; Port int}{
//	        Host: "localhost",
//	        Port: 5432,
//	    },
//	}
//	cfg, err := gokart.LoadConfigWithDefaults(defaults, "config.yaml")
func LoadConfigWithDefaults[T any](defaults T, paths ...string) (T, error) {
	v := viper.New()

	// Enable automatic environment variable binding
	v.AutomaticEnv()

	// Replace . with _ in environment variables (e.g., db.host → DB_HOST)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try each config path in order
	var configFound bool
	for _, path := range paths {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err == nil {
			configFound = true
			break
		}
	}

	// If no config file found but paths were provided, return error
	if !configFound && len(paths) > 0 {
		return defaults, fmt.Errorf("no config file found in paths: %v", paths)
	}

	// Unmarshal into result type
	var result T
	if err := v.Unmarshal(&result); err != nil {
		return defaults, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return result, nil
}
