package main

import (
	"os"
	"path/filepath"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

const logo = `
   ____       _  __          _
  / ___| ___ | |/ /__ _ _ __| |_
 | |  _ / _ \| ' // _' | '__| __|
 | |_| | (_) | . \ (_| | |  | |_
  \____|\___/|_|\_\__,_|_|   \__|
`

func main() {
	app := cli.NewApp("gokart", "0.1.0").
		WithDescription("Scaffold Go service projects").
		WithLongDescription(logo + `
gokart new <name> [flags]

  --sqlite     SQLite database (modernc.org/sqlite)
  --postgres   PostgreSQL pool (pgx/v5)
  --ai         OpenAI client (openai-go/v3)
  --flat       Single main.go (no internal/)
  --module     Custom module path`)

	newCmd := cli.CommandWithArgs("new <project-name>", "Create a new GoKart project", 1,
		func(cmd *cobra.Command, args []string) error {
			projectArg := args[0]

			flat, _ := cmd.Flags().GetBool("flat")
			module, _ := cmd.Flags().GetString("module")
			sqlite, _ := cmd.Flags().GetBool("sqlite")
			postgres, _ := cmd.Flags().GetBool("postgres")
			ai, _ := cmd.Flags().GetBool("ai")

			projectName := filepath.Base(projectArg)

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
				if sqlite || postgres || ai {
					cli.Warning("--sqlite, --postgres, and --ai flags are ignored in flat mode")
				}
				cli.Info("Scaffolding flat project: %s", projectName)
				if err := ScaffoldFlat(targetDir, projectName, module); err != nil {
					return err
				}
			} else {
				cli.Info("Scaffolding structured project: %s", projectName)
				if err := ScaffoldStructured(targetDir, projectName, module, sqlite, postgres, ai); err != nil {
					return err
				}
			}

			cli.Success("Project created at %s", targetDir)
			cli.Dim("  cd %s && go mod tidy", projectName)
			return nil
		})

	newCmd.Long = `Create a new Go project with sensible defaults and optional integrations.

Structured mode (default) creates:
  myapp/
  ├── main.go                    # Entry point
  ├── internal/
  │   ├── commands/              # Cobra command definitions
  │   └── actions/               # Business logic
  └── go.mod

Flat mode creates a single main.go for quick scripts.`

	newCmd.Example = `  # Basic structured project
  gokart new myapi

  # With PostgreSQL and OpenAI
  gokart new myapi --postgres --ai

  # With SQLite for local-first CLI
  gokart new mycli --sqlite

  # Quick script (single main.go)
  gokart new script --flat

  # Custom module path
  gokart new myapi --module github.com/myorg/myapi`

	newCmd.Flags().Bool("flat", false, "Use flat structure (single main.go)")
	newCmd.Flags().String("module", "", "Go module path (defaults to project name)")
	newCmd.Flags().Bool("sqlite", false, "Include SQLite database wiring (modernc.org/sqlite)")
	newCmd.Flags().Bool("postgres", false, "Include PostgreSQL connection pool (pgx/v5)")
	newCmd.Flags().Bool("ai", false, "Include OpenAI client (openai-go/v3)")

	app.AddCommand(newCmd)

	// Hide completion command from help
	for _, cmd := range app.Root().Commands() {
		if cmd.Name() == "completion" {
			cmd.Hidden = true
		}
		cli.SetStyledHelp(cmd)
	}

	// Minimal root help - just show Long description
	app.Root().SetHelpTemplate(`{{.Long}}

  gokart new myapp
  gokart new myapp --postgres --ai

  gokart new --help    Full options
`)

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
