// Package cli provides Cobra application construction for established
// ecosystem-style CLIs. For terminal presentation (styles, tables,
// spinners, progress), depend on lipgloss, bubbles, or lipgloss/table
// directly: this module deliberately does not wrap them.
package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// App builds CLI applications with sensible defaults.
type App struct {
	name        string
	version     string
	description string
	root        *cobra.Command
	viper       *viper.Viper
	configFile  string
	configName  string
	envPrefix   string
}

// NewApp creates a new CLI application builder.
//
// Example:
//
//	app := cli.NewApp("myapp", "1.0.0").
//	    WithDescription("My application").
//	    WithConfig("config.yaml").
//	    WithEnvPrefix("MYAPP")
//
//	app.AddCommand(serveCmd)
//	app.AddCommand(migrateCmd)
//
//	if err := app.Run(); err != nil {
//	    os.Exit(1)
//	}
func NewApp(name, version string) *App {
	app := &App{
		name:    name,
		version: version,
		viper:   viper.New(),
	}

	app.root = &cobra.Command{
		Use:     name,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return app.initConfig()
		},
	}

	return app
}

// WithDescription sets the app description.
func (a *App) WithDescription(desc string) *App {
	a.description = desc
	a.root.Short = desc
	return a
}

// WithLongDescription sets detailed description.
func (a *App) WithLongDescription(long string) *App {
	a.root.Long = long
	return a
}

// WithConfig sets the config file path.
func (a *App) WithConfig(path string) *App {
	a.configFile = path
	return a
}

// WithConfigName sets the config file name (without extension) to search for.
func (a *App) WithConfigName(name string) *App {
	a.configName = name
	return a
}

// WithEnvPrefix sets the environment variable prefix.
func (a *App) WithEnvPrefix(prefix string) *App {
	a.envPrefix = prefix
	a.viper.SetEnvPrefix(prefix)
	a.viper.AutomaticEnv()
	a.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	return a
}

// WithStandardFlags adds common flags (config, verbose, quiet).
func (a *App) WithStandardFlags() *App {
	flags := a.root.PersistentFlags()
	flags.StringVar(&a.configFile, "config", "", "config file path")
	flags.BoolP("verbose", "v", false, "verbose output")
	flags.BoolP("quiet", "q", false, "quiet output (errors only)")

	a.viper.BindPFlag("verbose", flags.Lookup("verbose"))
	a.viper.BindPFlag("quiet", flags.Lookup("quiet"))

	return a
}

// AddCommand adds a subcommand.
func (a *App) AddCommand(cmd *cobra.Command) *App {
	a.root.AddCommand(cmd)
	return a
}

// Root returns the root cobra command for advanced customization.
func (a *App) Root() *cobra.Command {
	return a.root
}

// Viper returns the viper instance for config access.
func (a *App) Viper() *viper.Viper {
	return a.viper
}

// Run executes the CLI application.
func (a *App) Run() error {
	return a.root.Execute()
}

// RunWithArgs executes with specific arguments (useful for testing).
func (a *App) RunWithArgs(args []string) error {
	a.root.SetArgs(args)
	return a.root.Execute()
}

// initConfig loads configuration from file and environment.
func (a *App) initConfig() error {
	if a.configFile != "" {
		a.viper.SetConfigFile(a.configFile)
	} else if a.configName != "" {
		a.viper.SetConfigName(a.configName)
		a.viper.SetConfigType("yaml")
		a.viper.AddConfigPath(".")
		if configDir, err := os.UserConfigDir(); err == nil {
			a.viper.AddConfigPath(filepath.Join(configDir, a.name))
		}
		a.viper.AddConfigPath("/etc/" + a.name)
	}

	if err := a.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

// Command creates a new cobra command with common setup.
//
// Example:
//
//	cmd := cli.Command("serve", "Start the server", func(cmd *cobra.Command, args []string) error {
//	    return server.Run()
//	})
func Command(use, short string, run func(cmd *cobra.Command, args []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE:  run,
	}
}

// CommandWithArgs creates a command that requires positional arguments.
func CommandWithArgs(use, short string, nArgs int, run func(cmd *cobra.Command, args []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(nArgs),
		RunE:  run,
	}
}

// Group creates a command group (no run function, just subcommands).
func Group(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
	}
}
