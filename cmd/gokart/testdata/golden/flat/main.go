package main

import (
	"os"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

// version is set via ldflags: -X main.version=v1.0.0
var version = "dev"

func main() {
	app := cli.NewApp("demo", version).
		WithDescription("demo - a GoKart CLI application")
	greetCmd := cli.Command("greet", "Greet someone", func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		loud, _ := cmd.Flags().GetBool("loud")

		msg := "Hello, " + name
		if loud {
			msg = "HELLO, " + name + "!"
		}

		cli.Success("%s", msg)
		return nil
	})
	greetCmd.Flags().StringP("name", "n", "World", "Name to greet")
	greetCmd.Flags().BoolP("loud", "l", false, "Greet loudly")

	app.AddCommand(greetCmd)

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
