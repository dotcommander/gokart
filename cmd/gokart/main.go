package main

import (
	"os"
	"path/filepath"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

func main() {
	app := cli.NewApp("gokart", "0.1.0").
		WithDescription("GoKart CLI - scaffold Go service projects")

	newCmd := cli.CommandWithArgs("new <project-name>", "Create a new GoKart project", 1,
		func(cmd *cobra.Command, args []string) error {
			projectArg := args[0]

			flat, _ := cmd.Flags().GetBool("flat")
			module, _ := cmd.Flags().GetString("module")
			sqlite, _ := cmd.Flags().GetBool("sqlite")
			ai, _ := cmd.Flags().GetBool("ai")

			// Extract name from path (e.g., /tmp/foo -> foo)
			projectName := filepath.Base(projectArg)

			// Determine target directory
			var targetDir string
			if filepath.IsAbs(projectArg) {
				targetDir = projectArg
			} else {
				targetDir = filepath.Join(".", projectArg)
			}

			if module == "" {
				module = projectName
			}

			if flat {
				if sqlite || ai {
					cli.Warning("--sqlite and --ai flags are ignored in flat mode")
				}
				cli.Info("Scaffolding flat project: %s", projectName)
				if err := ScaffoldFlat(targetDir, projectName, module); err != nil {
					return err
				}
			} else {
				cli.Info("Scaffolding structured project: %s", projectName)
				if err := ScaffoldStructured(targetDir, projectName, module, sqlite, ai); err != nil {
					return err
				}
			}

			cli.Success("Project created at %s", targetDir)
			cli.Dim("  cd %s && go mod tidy", projectName)
			return nil
		})

	newCmd.Flags().Bool("flat", false, "Use flat structure (single main.go)")
	newCmd.Flags().String("module", "", "Go module path (default: project name)")
	newCmd.Flags().Bool("sqlite", false, "Include SQLite database wiring")
	newCmd.Flags().Bool("ai", false, "Include OpenAI client wiring")

	app.AddCommand(newCmd)

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
