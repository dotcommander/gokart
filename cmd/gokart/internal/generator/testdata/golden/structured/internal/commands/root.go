package commands

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
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
	return parsed.Run()
}
