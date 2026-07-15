package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
)

var version = "dev"

type CLI struct {
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
	Greet   GreetCommand     `cmd:"" help:"Greet someone."`
}

type GreetCommand struct {
	Name string `short:"n" default:"World" help:"Name to greet."`
	Loud bool   `short:"l" help:"Greet loudly."`
}

func (c *GreetCommand) Run(kctx *kong.Context) error {
	msg := "Hello, " + c.Name
	if c.Loud {
		msg = "HELLO, " + c.Name + "!"
	}
	_, err := fmt.Fprintln(kctx.Stdout, msg)
	return err
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
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
	return parsed.Run()
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
