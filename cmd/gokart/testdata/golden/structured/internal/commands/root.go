package commands

import (
	"github.com/dotcommander/gokart/cli"
)

// Execute runs the CLI application.
func Execute(version string) error {
	cliApp := cli.NewApp("demo", version).
		WithDescription("demo CLI")
	cliApp.AddCommand(NewGreetCmd())

	return cliApp.Run()
}
