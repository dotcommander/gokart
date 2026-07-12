package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentedSQLiteNewcomerFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("exercises a generated module with go commands")
	}

	dir := t.TempDir()
	_, err := ScaffoldStructured(dir, "counter", "example.com/counter", true, false, false, false, false, true, ApplyOptions{ExistingFilePolicy: ExistingFilePolicyOverwrite})
	if err != nil {
		t.Fatal(err)
	}

	writeTutorialFile(t, dir, "migrations/00001_create_counter.sql", tutorialMigration)
	writeTutorialFile(t, dir, "internal/actions/counter.go", tutorialCounterAction)
	writeTutorialFile(t, dir, "internal/commands/counter.go", tutorialCounterCommand)

	rootPath := filepath.Join(dir, "internal/commands/root.go")
	root := readTutorialFile(t, rootPath)
	root = strings.Replace(root, "\t// Chain with existing PersistentPreRunE (config init)", "\tcliApp.Viper().SetDefault(app.AppConfigKeyDBPath, \"counter.db\")\n\n\t// Chain with existing PersistentPreRunE (config init)", 1)
	root = strings.Replace(root, "\tcliApp.AddCommand(NewGreetCmd(func() *app.Context {\n\t\treturn appCtx\n\t}))", "\tgetAppContext := func() *app.Context { return appCtx }\n\tcliApp.AddCommand(NewGreetCmd(getAppContext))\n\tcliApp.AddCommand(NewCounterCmd(getAppContext))", 1)
	writeTutorialFile(t, dir, "internal/commands/root.go", root)

	contextPath := filepath.Join(dir, "internal/app/context.go")
	appContext := readTutorialFile(t, contextPath)
	appContext = strings.Replace(appContext, "\t\"database/sql\"", "\t\"database/sql\"\n\t\"fmt\"", 1)
	appContext = strings.Replace(appContext, "\t\"github.com/dotcommander/gokart/sqlite\"", "\t\"github.com/dotcommander/gokart/migrate\"\n\t\"github.com/dotcommander/gokart/sqlite\"", 1)
	appContext = strings.Replace(appContext, "\tappCtx.DB = db", "\tif err := migrate.Up(ctx, db, migrate.Config{Dir: \"migrations\", Dialect: \"sqlite3\"}); err != nil {\n\t\t_ = db.Close()\n\t\treturn nil, fmt.Errorf(\"migrate database: %w\", err)\n\t}\n\tappCtx.DB = db", 1)
	writeTutorialFile(t, dir, "internal/app/context.go", appContext)

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for module, local := range map[string]string{
		"github.com/dotcommander/gokart":         repoRoot,
		"github.com/dotcommander/gokart/cli":     filepath.Join(repoRoot, "cli"),
		"github.com/dotcommander/gokart/sqlite":  filepath.Join(repoRoot, "sqlite"),
		"github.com/dotcommander/gokart/migrate": filepath.Join(repoRoot, "migrate"),
	} {
		runTutorialCommand(t, dir, "go", "mod", "edit", "-replace="+module+"="+local)
	}
	writeTutorialFile(t, dir, "go.sum", readTutorialFile(t, filepath.Join(repoRoot, "migrate", "go.sum")))
	runTutorialCommand(t, dir, "go", "mod", "edit", "-require=github.com/dotcommander/gokart/migrate@v0.11.0")
	runTutorialCommand(t, dir, "go", "mod", "tidy")
	runTutorialCommand(t, dir, "go", "test", "./...")
	if got := runTutorialCommand(t, dir, "go", "run", "./cmd", "greet", "--name", "Ada"); !strings.Contains(got, "Hello, Ada") {
		t.Fatalf("greet output = %q", got)
	}
	if got := runTutorialCommand(t, dir, "go", "run", "./cmd", "counter", "build", "--by", "3"); strings.TrimSpace(got) != "3" {
		t.Fatalf("counter output = %q, want 3", got)
	}
}

func writeTutorialFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTutorialFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func runTutorialCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOCACHE="+filepath.Join(os.TempDir(), "gokart-newcomer-go-cache"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

const tutorialMigration = `-- +goose Up
CREATE TABLE counters (name TEXT PRIMARY KEY, value INTEGER NOT NULL DEFAULT 0);
-- +goose Down
DROP TABLE counters;
`

const tutorialCounterAction = `package actions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dotcommander/gokart/sqlite"
)

func Increment(ctx context.Context, db *sql.DB, name string, by int64) (int64, error) {
	var value int64
	err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, ` + "`" + `INSERT INTO counters(name, value) VALUES (?, ?)
			ON CONFLICT(name) DO UPDATE SET value = value + excluded.value` + "`" + `, name, by)
		if err != nil { return fmt.Errorf("increment counter: %w", err) }
		return tx.QueryRowContext(ctx, ` + "`" + `SELECT value FROM counters WHERE name = ?` + "`" + `, name).Scan(&value)
	})
	return value, err
}
`

const tutorialCounterCommand = `package commands

import (
	"fmt"

	"example.com/counter/internal/actions"
	"example.com/counter/internal/app"
	"github.com/spf13/cobra"
)

func NewCounterCmd(getContext func() *app.Context) *cobra.Command {
	var by int64
	cmd := &cobra.Command{
		Use: "counter [name]", Short: "Increment a persistent counter", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			if len(args) == 1 { name = args[0] }
			value, err := actions.Increment(cmd.Context(), getContext().DB, name, by)
			if err != nil { return err }
			_, err = fmt.Fprintln(cmd.OutOrStdout(), value)
			return err
		},
	}
	cmd.Flags().Int64Var(&by, "by", 1, "amount to add")
	return cmd
}
`
