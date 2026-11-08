//go:build ignore

// Example: Kong application lifecycle with gokart/kongapp.
//
// The command tree, dependencies, signal handling, and exit policy stay
// caller-owned. kongapp only provides the parse → initialize → run →
// cleanup shell around your Kong tree.
//
// Run with: go run docs/examples/kongapp/main.go greet --name Ada
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/alecthomas/kong"
	"github.com/dotcommander/gokart/kongapp"
)

type CLI struct {
	Greet GreetCommand `cmd:"" help:"Greet someone."`
}

type GreetCommand struct {
	Name string `help:"Who to greet."`
}

func (c *GreetCommand) Run(ctx context.Context, streams kongapp.Streams, svc *greeter) error {
	_, err := fmt.Fprintf(streams.Out, "%s\n", svc.Greet(c.Name))
	return err
}

type greeter struct{}

func (g *greeter) Greet(name string) string {
	if name == "" {
		name = "World"
	}
	return "Hello, " + name + "!"
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var commands CLI
	err := kongapp.Execute(ctx, os.Args[1:], kongapp.Streams{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}, &commands, kongapp.Config[CLI]{
		Name:        "example",
		Description: "Example application",
		Options:     []kong.Option{kong.Bind(&greeter{})},
		Initialize: func(ctx context.Context, parsed *kong.Context, tree *CLI, args []string) (kongapp.Cleanup, error) {
			// Runs only after a command parses. Help/version never reach here.
			fmt.Fprintln(os.Stderr, "initializing…")
			return func() error {
				fmt.Fprintln(os.Stderr, "cleaning up…")
				return nil
			}, nil
		},
	})
	if err != nil {
		// main owns the exit policy; kongapp only returns the error.
		log.Fatal(err)
	}
}
