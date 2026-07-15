package commands

import (
	"context"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/example/demo/internal/app"
)

type CLI struct {
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
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
	if err := app.EnsureConfigDir(); err != nil {
		return err
	}
	return parsed.Run()
}
