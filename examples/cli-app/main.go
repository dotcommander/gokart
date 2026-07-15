// Example: focused typed CLI application using Kong.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Greet GreetCommand `cmd:"" help:"Greet someone."`
	Slow  SlowCommand  `cmd:"" help:"Do something slow."`
	Users UsersCommand `cmd:"" help:"List users."`
}

type GreetCommand struct {
	Name string `short:"n" default:"World" help:"Name to greet."`
}

func (c *GreetCommand) Run(ctx *kong.Context) error {
	_, err := fmt.Fprintf(ctx.Stdout, "Hello, %s!\n", c.Name)
	return err
}

type SlowCommand struct{}

func (SlowCommand) Run(ctx *kong.Context) error {
	if _, err := fmt.Fprintln(ctx.Stdout, "Processing..."); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

type UsersCommand struct{}

func (UsersCommand) Run(ctx *kong.Context) error {
	_, err := fmt.Fprintln(ctx.Stdout, "ID\tName\tRole\n1\tAlice\tAdmin\n2\tBob\tUser\n3\tCarol\tUser")
	return err
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli, kong.Name("greeter"), kong.Description("A friendly greeting CLI"), kong.UsageOnError(), kong.Writers(os.Stdout, os.Stderr))
	if err := ctx.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
