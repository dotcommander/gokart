package app

import (
	"github.com/dotcommander/gokart"
)

// ConfigDir returns the demo config directory (created if missing),
// using the platform-aware location from the root gokart module.
func ConfigDir() (string, error) {
	return gokart.ConfigDir("demo")
}

// EnsureConfigDir creates the config directory and writes the default
// config.yaml if it does not already exist.
func EnsureConfigDir() error {
	return gokart.EnsureConfigDir("demo", []byte(defaultConfig))
}

const defaultConfig = `# demo configuration
# Run: demo --help
`
