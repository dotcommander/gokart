package commands

import (
	"fmt"
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
			fmt.Fprintf(cmd.ErrOrStderr(), "greet failed: %v\n", err)
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), result)
		return nil
	})

	cmd.Flags().StringP("name", "n", "World", "Name to greet")
	cmd.Flags().BoolP("loud", "l", false, "Greet loudly")

	return cmd
}