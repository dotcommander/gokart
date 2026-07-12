package commands

import (
	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
	"github.com/example/demo/internal/app"
)

// Execute runs the CLI application.
func Execute(version string) error {
	cliApp := cli.NewApp("demo", version).
		WithDescription("demo CLI")

	// Chain with existing PersistentPreRunE (config init)
	origPreRun := cliApp.Root().PersistentPreRunE
	cliApp.Root().PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if origPreRun != nil {
			if err := origPreRun(cmd, args); err != nil {
				return err
			}
		}
		return app.EnsureConfigDir()
	}
	cliApp.AddCommand(NewGreetCmd())

	return cliApp.Run()
}
