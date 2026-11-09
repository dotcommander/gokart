package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/dotcommander/gokart/cli"
)

func quietApp(t *testing.T, name string) (*cli.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	app := cli.NewApp(name, "1.0.0")
	var stdout, stderr bytes.Buffer
	app.Root().SetOut(&stdout)
	app.Root().SetErr(&stderr)
	app.Root().SilenceUsage = true
	app.Root().SilenceErrors = true
	return app, &stdout, &stderr
}

func TestNewApp_Defaults(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	if app.Root() == nil {
		t.Fatal("Root() returned nil")
	}
	if app.Root().Use != "myapp" {
		t.Errorf("root Use = %q, want %q", app.Root().Use, "myapp")
	}
	if app.Root().Version != "1.0.0" {
		t.Errorf("root Version = %q, want %q", app.Root().Version, "1.0.0")
	}
	if app.Viper() == nil {
		t.Error("Viper() returned nil")
	}
}

func TestBuilderChainsReturnApp(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	if got := app.WithDescription("desc"); got != app {
		t.Error("WithDescription must return the same App for chaining")
	}
	if app.Root().Short != "desc" {
		t.Errorf("root Short = %q, want %q", app.Root().Short, "desc")
	}
	if got := app.WithLongDescription("long"); got != app {
		t.Error("WithLongDescription must return the same App")
	}
	if app.Root().Long != "long" {
		t.Errorf("root Long = %q, want %q", app.Root().Long, "long")
	}
	if got := app.WithConfig("cfg.yaml"); got != app {
		t.Error("WithConfig must return the same App")
	}
	if got := app.WithConfigName("myapp"); got != app {
		t.Error("WithConfigName must return the same App")
	}
	if got := app.WithStandardFlags(); got != app {
		t.Error("WithStandardFlags must return the same App")
	}
	if got := app.AddCommand(cli.Command("x", "x", func(*cobra.Command, []string) error { return nil })); got != app {
		t.Error("AddCommand must return the same App")
	}
}

func TestWithStandardFlagsRegistersFlags(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	app.WithStandardFlags()
	for _, want := range []string{"config", "verbose", "quiet"} {
		if app.Root().PersistentFlags().Lookup(want) == nil {
			t.Errorf("flag %q not registered", want)
		}
	}
}

func TestRunWithArgsDispatchesCommand(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	run := false
	app.AddCommand(cli.Command("hello", "hello", func(cmd *cobra.Command, args []string) error {
		run = true
		return nil
	}))
	if err := app.RunWithArgs([]string{"hello"}); err != nil {
		t.Fatalf("RunWithArgs: %v", err)
	}
	if !run {
		t.Error("command RunE was not invoked")
	}
}

func TestCommandWithArgsRequiresExactCount(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	app.AddCommand(cli.CommandWithArgs("pair", "pair", 2, func(cmd *cobra.Command, args []string) error {
		return nil
	}))
	if err := app.RunWithArgs([]string{"pair", "only-one"}); err == nil {
		t.Error("expected error when arg count is wrong")
	}
	if err := app.RunWithArgs([]string{"pair", "a", "b"}); err != nil {
		t.Errorf("expected no error with exact args, got %v", err)
	}
}

func TestGroupCreatesContainer(t *testing.T) {
	t.Parallel()
	g := cli.Group("serve", "serve commands")
	if g.Use != "serve" || g.Short != "serve commands" {
		t.Errorf("Group = %v, want Use=serve Short=%q", g, "serve commands")
	}
}

// Runs RunWithArgs so initConfig (PersistentPreRunE) executes.
func runEchoViper(t *testing.T, app *cli.App, key string) (string, error) {
	t.Helper()
	var got string
	app.AddCommand(cli.Command("probe", "probe", func(cmd *cobra.Command, args []string) error {
		got = app.Viper().GetString(key)
		return nil
	}))
	err := app.RunWithArgs([]string{"probe"})
	return got, err
}

func TestInitConfig_LoadsExplicitFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(path, []byte("token: abc\n"), 0600); err != nil {
		t.Fatal(err)
	}
	app, _, _ := quietApp(t, "myapp")
	app.WithConfig(path)
	got, err := runEchoViper(t, app, "token")
	if err != nil {
		t.Fatalf("RunWithArgs: %v", err)
	}
	if got != "abc" {
		t.Errorf("viper token = %q, want %q", got, "abc")
	}
}

func TestInitConfig_ExplicitMissingFileErrors(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	app.WithConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if _, err := runEchoViper(t, app, "token"); err == nil {
		t.Fatal("explicit missing config file must error")
	}
}

func TestInitConfig_NamedMissingFileTolerated(t *testing.T) {
	t.Parallel()
	app, _, _ := quietApp(t, "myapp")
	app.WithConfigName("definitely-not-present-xyz")
	if _, err := runEchoViper(t, app, "token"); err != nil {
		t.Fatalf("searched missing config file must be tolerated, got %v", err)
	}
}

func TestInitConfig_MalformedExplicitFileErrors(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte(": : :"), 0600); err != nil {
		t.Fatal(err)
	}
	app, _, _ := quietApp(t, "myapp")
	app.WithConfig(path)
	if _, err := runEchoViper(t, app, "token"); err == nil {
		t.Fatal("malformed explicit config must error")
	}
}

func TestCaptureInputUsesEditorEnv(t *testing.T) {
	// Serial: reads process environment.
	t.Setenv("EDITOR", "true")
	got, err := cli.CaptureInput("hello", "txt")
	if err != nil {
		t.Fatalf("CaptureInput: %v", err)
	}
	if got != "hello" {
		t.Errorf("CaptureInput = %q, want %q", got, "hello")
	}
}

// Serial: mutates process environment.
func TestWithEnvPrefixReadsEnvironment(t *testing.T) {
	t.Setenv("MYAPP_TOKEN", "xyz")
	app, _, _ := quietApp(t, "myapp")
	app.WithEnvPrefix("MYAPP")
	if got := app.Viper().GetString("token"); got != "xyz" {
		t.Errorf("env token = %q, want %q", got, "xyz")
	}
}
