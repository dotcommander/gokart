package generator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recordingRunner struct {
	commands []Command
	failAt   int
}

func (r *recordingRunner) Run(_ context.Context, command Command) error {
	r.commands = append(r.commands, command)
	if r.failAt > 0 && len(r.commands) == r.failAt {
		return errors.New("runner failed")
	}
	return nil
}

func TestCreatePreparesDependenciesWhenVerificationDisabled(t *testing.T) {
	t.Parallel()
	runner := &recordingRunner{}
	service := New(Dependencies{Runner: runner})
	target := filepath.Join(t.TempDir(), "demo")
	result, err := service.Create(context.Background(), CreateRequest{
		Args: []string{target}, NoVerify: true, ConfigScope: configScopeAuto,
	}, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := commandArgs(runner.commands), []string{"go get github.com/alecthomas/kong@v1.15.0", "go mod tidy"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands=%v want=%v", got, want)
	}
	if result.VerifyRan {
		t.Fatal("verification unexpectedly ran")
	}
	if got, want := strings.Join(result.NextArgs, " "), "build -o demo ."; got != want {
		t.Fatalf("next args=%q want=%q", got, want)
	}
}

func TestCreateRunsTestsAndBuildAfterSingleDependencyPreparation(t *testing.T) {
	t.Parallel()
	runner := &recordingRunner{}
	service := New(Dependencies{Runner: runner})
	target := filepath.Join(t.TempDir(), "demo")
	result, err := service.Create(context.Background(), CreateRequest{
		Args: []string{target}, Example: true, ConfigScope: configScopeAuto,
	}, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"go get github.com/alecthomas/kong@v1.15.0", "go mod tidy", "go test ./...", "go build ./..."}
	if got := commandArgs(runner.commands); !reflect.DeepEqual(got, want) {
		t.Fatalf("commands=%v want=%v", got, want)
	}
	if !result.VerifyRan || !result.VerifyPassed || len(result.Checks) != 4 {
		t.Fatalf("result=%+v", result)
	}
}

func TestCreateDependencyFailureIsPartialScaffoldFailure(t *testing.T) {
	t.Parallel()
	runner := &recordingRunner{failAt: 1}
	service := New(Dependencies{Runner: runner})
	target := filepath.Join(t.TempDir(), "demo")
	result, err := service.Create(context.Background(), CreateRequest{
		Args: []string{target}, NoVerify: true, ConfigScope: configScopeAuto,
	}, Runtime{})
	var op *OperationError
	if !errors.As(err, &op) || op.Kind != ErrorScaffoldFailed || !op.Partial {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(err.Error(), "recover with: cd ") || len(result.Checks) != 1 || result.Checks[0].Status != "failed" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestVerifyOnlyTidiesTestsAndBuilds(t *testing.T) {
	t.Parallel()
	runner := &recordingRunner{}
	service := New(Dependencies{Runner: runner})
	target := t.TempDir()
	result, err := service.Create(context.Background(), CreateRequest{
		Args: []string{target}, VerifyOnly: true, ConfigScope: configScopeAuto,
	}, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"go mod tidy", "go test ./...", "go build ./..."}
	if got := commandArgs(runner.commands); !reflect.DeepEqual(got, want) || !result.VerifyPassed {
		t.Fatalf("commands=%v result=%+v", got, result)
	}
}

func TestStructuredExampleIsImmediatelyAddCompatible(t *testing.T) {
	t.Parallel()
	runner := &recordingRunner{}
	service := New(Dependencies{Runner: runner})
	target := filepath.Join(t.TempDir(), "demo")
	result, err := service.Create(context.Background(), CreateRequest{
		Args: []string{target}, Structured: true, Example: true, NoVerify: true, ConfigScope: configScopeAuto,
	}, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.WriteManifest {
		t.Fatal("structured scaffold did not write a manifest")
	}
	add, err := service.Add(context.Background(), AddRequest{Dir: target, Integrations: []string{"sqlite"}, DryRun: true}, Runtime{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(add.Added, []string{"sqlite"}) {
		t.Fatalf("add result=%+v", add)
	}
}

func TestFlatScaffoldNeverWritesManifest(t *testing.T) {
	t.Parallel()
	for _, useGlobal := range []bool{false, true} {
		dir := t.TempDir()
		if _, err := ScaffoldFlat(dir, "demo", "example.com/demo", useGlobal, true, ApplyOptions{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, scaffoldManifestPath)); !os.IsNotExist(err) {
			t.Fatalf("useGlobal=%v manifest stat error=%v", useGlobal, err)
		}
	}
}

func commandArgs(commands []Command) []string {
	result := make([]string, 0, len(commands))
	for _, command := range commands {
		result = append(result, command.Name+" "+strings.Join(command.Args, " "))
	}
	return result
}
