package main

import (
	"context"
	"fmt"
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
	fmt.Fprintln(kctx.Stdout, msg)
	return nil
}

func run(ctx context.Context) error {
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

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
