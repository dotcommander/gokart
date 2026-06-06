package commands

import (
	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
	"github.com/example/demo/internal/app"
)

// Execute runs the CLI application.
func Execute(version string) error {
	cliApp := cli.NewApp("demo", version).
		WithDescription("demo CLI").
		WithEnvPrefix("DEMO").
		WithStandardFlags()

	// Chain with existing PersistentPreRunE (config init)
	var appCtx *app.Context
	origPreRun := cliApp.Root().PersistentPreRunE
	cliApp.Root().PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if origPreRun != nil {
			if err := origPreRun(cmd, args); err != nil {
				return err
			}
		}
		var err error
		appCtx, err = app.New(cmd.Context(), "demo", cliApp.Viper())
		return err
	}

	// Cleanup on exit
	defer func() {
		if appCtx != nil {
			appCtx.Close()
		}
	}()
	cliApp.AddCommand(NewGreetCmd(func() *app.Context {
		return appCtx
	}))

	return cliApp.Run()
}
