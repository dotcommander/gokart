package commands

import (
	"context"
	"errors"
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

func Execute(ctx context.Context, version string) error {
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("demo"), kong.Description("demo CLI"), kong.Vars{"version": version}, kong.UsageOnError(), kong.BindTo(ctx, (*context.Context)(nil)))
	if err != nil {
		return err
	}
	parsed, err := parser.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	if len(os.Args) == 1 {
		return parsed.PrintUsage(false)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}
	userConfigDir, _ := os.UserConfigDir()
	v, err := loadConfig(cli, workingDir, userConfigDir, "/etc")
	if err != nil {
		return err
	}
	appCtx, err := app.New(ctx, "demo", v)
	if err != nil {
		return err
	}
	defer appCtx.Close()
	return parsed.Run(appCtx)
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
