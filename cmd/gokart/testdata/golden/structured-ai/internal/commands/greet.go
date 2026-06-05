package commands

import (
	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"

	"github.com/example/demo/internal/actions"
	"github.com/example/demo/internal/app"
)

// NewGreetCmd creates the greet command.
func NewGreetCmd(getAppContext func() *app.Context) *cobra.Command {
	cmd := cli.Command("greet", "Greet someone", func(cmd *cobra.Command, args []string) error {
		name := cmd.Flag("name").Value.String()
		loud, _ := cmd.Flags().GetBool("loud")

		input := actions.GreetInput{
			Name: name,
			Loud: loud,
		}
		var appCtx *app.Context
		if getAppContext != nil {
			appCtx = getAppContext()
		}

		result, err := actions.Greet(appCtx, input)
		if err != nil {
			cli.Error("greet failed: %v", err)
			return err
		}

		cli.Success("%s", result)
		return nil
	})

	cmd.Flags().StringP("name", "n", "World", "Name to greet")
	cmd.Flags().BoolP("loud", "l", false, "Greet loudly")

	return cmd
}