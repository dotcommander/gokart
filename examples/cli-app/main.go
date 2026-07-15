// Example: a deterministic, testable TV-guide CLI using Kong.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
)

var version = "dev"

type CLI struct {
	Version kong.VersionFlag `name:"version" help:"Print version information and quit."`
	Now     NowCommand       `cmd:"" help:"Show programs airing now."`
}

type Program struct {
	Channel string
	Starts  string
	Title   string
}

var schedule = []Program{
	{Channel: "WGBH", Starts: "8:00 PM", Title: "Nature: Wild Coast"},
	{Channel: "WGBX", Starts: "8:00 PM", Title: "The Great British Bake Off"},
	{Channel: "WCVB", Starts: "8:30 PM", Title: "Chronicle"},
}

type NowCommand struct {
	Channel string `short:"c" help:"Only show this channel."`
}

func (c *NowCommand) Run(kctx *kong.Context) error {
	programs := schedule
	if c.Channel != "" {
		programs = programsForChannel(schedule, c.Channel)
		if len(programs) == 0 {
			return fmt.Errorf("no programs found for channel %q", c.Channel)
		}
	}
	return writePrograms(kctx.Stdout, programs)
}

func programsForChannel(programs []Program, channel string) []Program {
	selected := make([]Program, 0, len(programs))
	for _, program := range programs {
		if strings.EqualFold(program.Channel, channel) {
			selected = append(selected, program)
		}
	}
	return selected
}

func writePrograms(w io.Writer, programs []Program) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "CHANNEL\tSTARTS\tPROGRAM"); err != nil {
		return err
	}
	for _, program := range programs {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", program.Channel, program.Starts, program.Title); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	var cli CLI
	parser, err := kong.New(
		&cli,
		kong.Name("tvguide"),
		kong.Description("Show deterministic TV listings."),
		kong.Vars{"version": version},
		kong.Writers(stdout, stderr),
		kong.BindTo(ctx, (*context.Context)(nil)),
	)
	if err != nil {
		return err
	}
	parsed, err := parser.Parse(args)
	if err != nil {
		var parseErr *kong.ParseError
		if len(args) == 0 && errors.As(err, &parseErr) {
			return parseErr.Context.PrintUsage(false)
		}
		return err
	}
	return parsed.Run()
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
