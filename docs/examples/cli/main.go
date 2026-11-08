//go:build ignore

// Example: Ecosystem-style Cobra CLI with gokart/cli.
//
// For new focused CLIs, prefer typed Kong command structs (see
// examples/cli-app) — gokart/cli is for established Cobra applications.
// Your main owns the exit status; commands return errors.
//
// Run with: go run docs/examples/cli/main.go greet --name Ada
package main

import (
	"fmt"
	"os"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

func main() {
	app := cli.NewApp("example", "1.0.0").
		WithDescription("Example CLI").
		WithStandardFlags()

	greet := cli.Command("greet", "Greet someone", func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = "World"
		}
		// Command output goes through Cobra so tests can capture it.
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Hello, %s!\n", name)
		return err
	})
	greet.Flags().String("name", "", "Who to greet.")
	app.AddCommand(greet)

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
