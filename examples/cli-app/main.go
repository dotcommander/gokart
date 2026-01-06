// Example: Minimal CLI application using gokart/cli.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

func main() {
	app := cli.NewApp("greeter", "1.0.0").
		WithDescription("A friendly greeting CLI")

	// Add greet command
	greetCmd := cli.Command("greet", "Greet someone", func(cmd *cobra.Command, args []string) error {
		name := cmd.Flag("name").Value.String()
		cli.Success("Hello, %s!", name)
		return nil
	})
	greetCmd.Flags().StringP("name", "n", "World", "Name to greet")
	app.AddCommand(greetCmd)

	// Add slow command with spinner
	app.AddCommand(cli.Command("slow", "Do something slow", func(cmd *cobra.Command, args []string) error {
		return cli.WithSpinner("Processing...", func() error {
			time.Sleep(2 * time.Second)
			return nil
		})
	}))

	// Add table command
	app.AddCommand(cli.Command("users", "List users", func(cmd *cobra.Command, args []string) error {
		t := cli.NewTable("ID", "Name", "Role")
		t.AddRow("1", "Alice", "Admin")
		t.AddRow("2", "Bob", "User")
		t.AddRow("3", "Carol", "User")
		t.Print()
		return nil
	}))

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
