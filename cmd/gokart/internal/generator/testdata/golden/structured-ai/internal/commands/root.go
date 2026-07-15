package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/example/demo/internal/app"
	"github.com/spf13/viper"
)

type CLI struct {
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
	Config  string           `type:"path" help:"Config file path."`
	Verbose *bool            `short:"v" help:"Enable verbose output."`
	Quiet   *bool            `short:"q" help:"Only print errors."`
	Greet   GreetCommand     `cmd:"" help:"Greet someone."`
}

// Execute runs the CLI using the process arguments and streams.
func Execute(ctx context.Context, version string) error {
	return execute(ctx, version, os.Args[1:], os.Stdout, os.Stderr)
}

func execute(ctx context.Context, version string, args []string, stdout, stderr io.Writer) error {
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("demo"), kong.Description("demo CLI"), kong.Vars{"version": version}, kong.Writers(stdout, stderr), kong.UsageOnError(), kong.BindTo(ctx, (*context.Context)(nil)))
	if err != nil {
		return err
	}
	if len(args) == 0 {
		usage, err := kong.Trace(parser, args)
		if err != nil {
			return err
		}
		return usage.PrintUsage(false)
	}
	parsed, err := parser.Parse(args)
	if err != nil {
		return err
	}
	var dependencies *app.Dependencies
	if err := parsed.BindSingletonProvider(func() (*app.Dependencies, error) {
		workingDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		userConfigDir, _ := os.UserConfigDir()
		v, err := loadConfig(cli, workingDir, userConfigDir, "/etc")
		if err != nil {
			return nil, err
		}
		dependencies, err = app.New(ctx, "demo", v)
		return dependencies, err
	}); err != nil {
		return err
	}
	runErr := parsed.Run()
	if dependencies == nil {
		return runErr
	}
	return errors.Join(runErr, dependencies.Close())
}

func loadConfig(cli CLI, workingDir, userConfigDir, systemConfigRoot string) (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvPrefix("DEMO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	if cli.Config != "" {
		v.SetConfigFile(cli.Config)
	} else {
		v.SetConfigName("demo")
		v.SetConfigType("yaml")
		v.AddConfigPath(workingDir)
		if userConfigDir != "" {
			v.AddConfigPath(filepath.Join(userConfigDir, "demo"))
		}
		if systemConfigRoot != "" {
			v.AddConfigPath(filepath.Join(systemConfigRoot, "demo"))
		}
	}
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if cli.Config != "" || !errors.As(err, &notFound) {
			return nil, err
		}
	}
	if cli.Verbose != nil {
		v.Set("verbose", *cli.Verbose)
	}
	if cli.Quiet != nil {
		v.Set("quiet", *cli.Quiet)
	}
	return v, nil
}
