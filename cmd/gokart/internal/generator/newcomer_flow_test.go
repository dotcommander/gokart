package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentedSQLiteNewcomerFlow(t *testing.T) {
	t.Parallel()
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
	writeTutorialFile(t, dir, "internal/commands/config_runtime_test.go", generatedConfigRuntimeTest)

	rootPath := filepath.Join(dir, "internal/commands/root.go")
	root := readTutorialFile(t, rootPath)
	root = strings.Replace(root, "\tGreet   GreetCommand", "\tCounter CounterCommand `cmd:\"\" help:\"Increment a persistent counter.\"`\n\tGreet   GreetCommand", 1)
	root = strings.Replace(root, "\tv := viper.New()", "\tv := viper.New()\n\tv.SetDefault(app.AppConfigKeyDBPath, \"counter.db\")", 1)
	writeTutorialFile(t, dir, "internal/commands/root.go", root)

	contextPath := filepath.Join(dir, "internal/app/context.go")
	appContext := readTutorialFile(t, contextPath)
	appContext = strings.Replace(appContext, "\t\"database/sql\"", "\t\"database/sql\"\n\t\"fmt\"", 1)
	appContext = strings.Replace(appContext, "\t\"github.com/dotcommander/gokart/sqlite\"", "\t\"github.com/dotcommander/gokart/migrate\"\n\t\"github.com/dotcommander/gokart/sqlite\"", 1)
	appContext = strings.Replace(appContext, "\tappCtx.DB = db", "\tif err := migrate.Up(ctx, db, migrate.Config{Dir: \"migrations\", Dialect: \"sqlite3\"}); err != nil {\n\t\t_ = db.Close()\n\t\treturn nil, fmt.Errorf(\"migrate database: %w\", err)\n\t}\n\tappCtx.DB = db", 1)
	writeTutorialFile(t, dir, "internal/app/context.go", appContext)

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for module, local := range map[string]string{
		"github.com/dotcommander/gokart":         repoRoot,
		"github.com/dotcommander/gokart/sqlite":  filepath.Join(repoRoot, "sqlite"),
		"github.com/dotcommander/gokart/migrate": filepath.Join(repoRoot, "migrate"),
	} {
		runTutorialGoCommand(t, dir, "mod", "edit", "-replace="+module+"="+local)
	}
	writeTutorialFile(t, dir, "go.sum", readTutorialFile(t, filepath.Join(repoRoot, "migrate", "go.sum")))
	runTutorialGoCommand(t, dir, "mod", "edit", "-require=github.com/dotcommander/gokart/migrate@v0.12.0")
	runTutorialGoCommand(t, dir, "mod", "tidy")
	runTutorialGoCommand(t, dir, "test", "./...")
	writeTutorialFile(t, dir, "counter.yaml", "db_path: discovered.db\n")
	if got := runTutorialGoCommand(t, dir, "run", "./cmd", "greet", "--name", "Ada"); !strings.Contains(got, "Hello, Ada") {
		t.Fatalf("greet output = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "discovered.db")); err != nil {
		t.Fatalf("implicit current-directory config was not loaded: %v", err)
	}
	writeTutorialFile(t, dir, "explicit.yaml", "db_path: explicit.db\n")
	runTutorialGoCommand(t, dir, "run", "./cmd", "greet", "--config", "explicit.yaml")
	if _, err := os.Stat(filepath.Join(dir, "explicit.db")); err != nil {
		t.Fatalf("explicit config was not loaded: %v", err)
	}
	if output, err := runTutorialGoCommandResult(dir, "run", "./cmd", "greet", "--config", "missing.yaml"); err == nil || !strings.Contains(output, "missing.yaml") {
		t.Fatalf("explicit missing config should fail actionably: err=%v output=%q", err, output)
	}
	if got := runTutorialGoCommand(t, dir, "run", "./cmd", "counter", "build", "--by", "3"); strings.TrimSpace(got) != "3" {
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

func runTutorialGoCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runTutorialGoCommandResult(dir, args...)
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runTutorialGoCommandResult(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOCACHE="+filepath.Join(os.TempDir(), "gokart-newcomer-go-cache"))
	out, err := cmd.CombinedOutput()
	return string(out), err
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
	"context"
	"fmt"
	"github.com/alecthomas/kong"

	"example.com/counter/internal/actions"
	"example.com/counter/internal/app"
)

type CounterCommand struct {
	Name string ` + "`arg:\"\" optional:\"\" default:\"default\"`" + `
	By int64 ` + "`default:\"1\"`" + `
}

func (c *CounterCommand) Run(ctx context.Context, kctx *kong.Context, appCtx *app.Context) error {
	value, err := actions.Increment(ctx, appCtx.DB, c.Name, c.By)
	if err != nil { return err }
	_, err = fmt.Fprintln(kctx.Stdout, value)
	return err
}
`

const generatedConfigRuntimeTest = `package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedConfigPrecedence(t *testing.T) {
	t.Setenv("COUNTER_DB_PATH", "")
	workingDir := t.TempDir()
	userRoot := t.TempDir()
	systemRoot := t.TempDir()

	v, err := loadConfig(CLI{}, workingDir, userRoot, systemRoot)
	if err != nil { t.Fatalf("absent implicit config: %v", err) }
	if got := v.GetString("db_path"); got != "counter.db" { t.Fatalf("absent implicit db_path=%q", got) }

	writeConfig(t, filepath.Join(systemRoot, "counter", "counter.yaml"), "db_path: system.db\n")
	v, err = loadConfig(CLI{}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetString("db_path") != "system.db" { t.Fatalf("system discovery: value=%q err=%v", v.GetString("db_path"), err) }

	writeConfig(t, filepath.Join(userRoot, "counter", "counter.yaml"), "db_path: user.db\n")
	v, err = loadConfig(CLI{}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetString("db_path") != "user.db" { t.Fatalf("user discovery: value=%q err=%v", v.GetString("db_path"), err) }

	writeConfig(t, filepath.Join(workingDir, "counter.yaml"), "db_path: cwd.db\nverbose: true\nquiet: true\n")
	v, err = loadConfig(CLI{}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetString("db_path") != "cwd.db" { t.Fatalf("cwd discovery: value=%q err=%v", v.GetString("db_path"), err) }

	t.Setenv("COUNTER_DB_PATH", "env.db")
	v, err = loadConfig(CLI{}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetString("db_path") != "env.db" { t.Fatalf("env precedence: value=%q err=%v", v.GetString("db_path"), err) }

	falseValue := false
	t.Setenv("COUNTER_VERBOSE", "true")
	t.Setenv("COUNTER_QUIET", "true")
	v, err = loadConfig(CLI{Verbose: &falseValue, Quiet: &falseValue}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetBool("verbose") || v.GetBool("quiet") { t.Fatalf("explicit false flags: verbose=%v quiet=%v err=%v", v.GetBool("verbose"), v.GetBool("quiet"), err) }

	explicit := filepath.Join(t.TempDir(), "explicit.yaml")
	writeConfig(t, explicit, "db_path: explicit.db\n")
	v, err = loadConfig(CLI{Config: explicit}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetString("db_path") != "env.db" { t.Fatalf("env over explicit config: value=%q err=%v", v.GetString("db_path"), err) }
	t.Setenv("COUNTER_DB_PATH", "")
	v, err = loadConfig(CLI{Config: explicit}, workingDir, userRoot, systemRoot)
	if err != nil || v.GetString("db_path") != "explicit.db" { t.Fatalf("explicit over discovery: value=%q err=%v", v.GetString("db_path"), err) }

	missing := filepath.Join(t.TempDir(), "missing.yaml")
	_, err = loadConfig(CLI{Config: missing}, workingDir, userRoot, systemRoot)
	if err == nil || !strings.Contains(err.Error(), "missing.yaml") { t.Fatalf("missing explicit config: %v", err) }
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(content), 0644); err != nil { t.Fatal(err) }
}
`
