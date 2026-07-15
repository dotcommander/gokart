package commands

import (
	"fmt"
	"github.com/alecthomas/kong"

	"github.com/example/demo/internal/actions"
	"github.com/example/demo/internal/app"
)

type GreetCommand struct {
	Name string `short:"n" default:"World" help:"Name to greet."`
	Loud bool   `short:"l" help:"Greet loudly."`
}

func (c *GreetCommand) Run(kctx *kong.Context, appCtx *app.Context) error {
	input := actions.GreetInput{Name: c.Name, Loud: c.Loud}
	result, err := actions.Greet(appCtx, input)
	if err != nil {
		return fmt.Errorf("greet failed: %w", err)
	}
	fmt.Fprintln(kctx.Stdout, result)
	return nil
}
