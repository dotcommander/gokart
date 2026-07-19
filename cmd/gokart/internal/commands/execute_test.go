package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

type fakeProjects struct {
	create func(generator.CreateRequest) (generator.CreateResult, error)
	add    func(generator.AddRequest) (generator.AddResult, error)
}

func (f fakeProjects) Create(_ context.Context, req generator.CreateRequest, _ generator.Runtime) (generator.CreateResult, error) {
	return f.create(req)
}

func (f fakeProjects) Add(_ context.Context, req generator.AddRequest, _ generator.Runtime) (generator.AddResult, error) {
	return f.add(req)
}

func testDependencies(stdout, stderr *bytes.Buffer, projects ProjectGenerator) Dependencies {
	return Dependencies{Projects: projects, Stdout: stdout, Stderr: stderr,
		Getwd:         func() (string, error) { return "/tmp/work", nil },
		UserConfigDir: func() (string, error) { return "/tmp/config", nil }, BinaryPath: "/tmp/gokart"}
}

func TestExecutePassesExplicitArgsAndLayoutFlags(t *testing.T) {
	t.Parallel()
	var got generator.CreateRequest
	projects := fakeProjects{create: func(req generator.CreateRequest) (generator.CreateResult, error) {
		got = req
		return generator.CreateResult{Mode: "structured", DryRun: true}, nil
	}, add: func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil }}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"new", "demo", "--structured", "--dry-run", "--json", "--no-verify"}, "v-test", testDependencies(&stdout, &stderr, projects))
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if !got.Structured || got.WorkingDir != "/tmp/work" || len(got.Args) != 1 || got.Args[0] != "demo" {
		t.Fatalf("request=%+v", got)
	}
	var out createOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v: %q", err, stdout.String())
	}
	if out.Mode != "structured" || out.Outcome != "success" {
		t.Fatalf("output=%+v", out)
	}
}

func TestExecuteJSONFailureIsSingleObjectWithoutHumanError(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{create: func(generator.CreateRequest) (generator.CreateResult, error) {
		return generator.CreateResult{}, &generator.OperationError{Kind: generator.ErrorInvalidArguments, Err: errors.New("bad flags")}
	}, add: func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil }}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"new", "demo", "--json", "--no-verify"}, "v-test", testDependencies(&stdout, &stderr, projects))
	if code != 2 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	var out createOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.ErrorCode != generator.ErrorInvalidArguments {
		t.Fatalf("output=%+v", out)
	}
}

func TestExecuteParseFailuresUseInvalidArgumentsExitCode(t *testing.T) {
	t.Parallel()

	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	for _, args := range [][]string{{"new"}, {"new", "demo", "--unknown"}} {
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), args, "v-test", testDependencies(&stdout, &stderr, projects)); code != 2 {
			t.Errorf("args=%v exit=%d, want 2; stderr=%q", args, code, stderr.String())
		}
	}
}

func TestExecuteConfigPreservesLegacyBehavior(t *testing.T) {
	t.Parallel()

	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	var stdout, stderr bytes.Buffer
	deps := testDependencies(&stdout, &stderr, projects)
	deps.UserConfigDir = func() (string, error) { return "/tmp/user-config", nil }

	if code := Execute(context.Background(), []string{configName}, "v-test", deps); code != 0 {
		t.Fatalf("config exit=%d stderr=%q", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: gokart config")) {
		t.Fatalf("config help=%q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Execute(context.Background(), []string{configName, "show"}, "v-test", deps); code != 0 {
		t.Fatalf("config show exit=%d stderr=%q", code, stderr.String())
	}
	want := "Version:     v-test\nConfig dir:  /tmp/user-config\nBinary:      /tmp/gokart\n"
	if stdout.String() != want {
		t.Fatalf("config show=%q, want %q", stdout.String(), want)
	}
}

func TestExecuteVersionEntrypointsPreserveLegacyOutput(t *testing.T) {
	t.Parallel()

	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	for _, args := range [][]string{{"--version"}, {"version"}} {
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), args, "v-test", testDependencies(&stdout, &stderr, projects)); code != 0 {
			t.Fatalf("args=%v exit=%d stderr=%q", args, code, stderr.String())
		}
		if want := "gokart version v-test\n"; stdout.String() != want {
			t.Fatalf("args=%v output=%q, want %q", args, stdout.String(), want)
		}
	}
}

func TestExecuteLegacyHelpEntrypointsSucceed(t *testing.T) {
	t.Parallel()

	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	for _, args := range [][]string{nil, {"--version=false"}} {
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), args, "v-test", testDependencies(&stdout, &stderr, projects)); code != 0 {
			t.Fatalf("args=%v exit=%d stderr=%q", args, code, stderr.String())
		}
		if !bytes.Contains(stdout.Bytes(), []byte("Usage: gokart")) {
			t.Fatalf("args=%v help=%q", args, stdout.String())
		}
	}
}

func TestExecuteConfigShowReportsUnavailableConfigDirectory(t *testing.T) {
	t.Parallel()

	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	var stdout, stderr bytes.Buffer
	deps := testDependencies(&stdout, &stderr, projects)
	deps.UserConfigDir = func() (string, error) { return "", errors.New("unsupported") }
	if code := Execute(context.Background(), []string{configName, "show"}, "v-test", deps); code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config dir:  (unavailable: unsupported)") {
		t.Fatalf("output=%q", stdout.String())
	}
}

func TestCreateOutputShellQuotesNextCommandDirectory(t *testing.T) {
	t.Parallel()

	out := createOutputFrom(generator.CreateResult{
		NextDir:     "/tmp/operator's project",
		NextCommand: "go",
		NextArgs:    []string{"build", "./..."},
	})
	if want := `cd '/tmp/operator'"'"'s project' && go build ./...`; out.NextCommand != want {
		t.Fatalf("next_command = %q, want %q", out.NextCommand, want)
	}
}

func TestExecuteAliasesAndRejectsCompletion(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{create: func(req generator.CreateRequest) (generator.CreateResult, error) {
		return generator.CreateResult{ProjectName: req.Args[0], DryRun: true}, nil
	}, add: func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil }}
	for _, alias := range []string{"new", "create", "init"} {
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), []string{alias, "demo", "--dry-run", "--no-verify"}, "v-test", testDependencies(&stdout, &stderr, projects)); code != 0 {
			t.Fatalf("%s exit=%d stderr=%q", alias, code, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"completion"}, "v-test", testDependencies(&stdout, &stderr, projects)); code == 0 {
		t.Fatal("completion unexpectedly accepted")
	}
}

func TestHelpSnapshots(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	for _, command := range []string{"root", "new", "add"} {
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			deps := testDependencies(&stdout, &stderr, projects)
			exec := &executor{ctx: context.Background(), version: "v-test", deps: deps}
			root := cli{}
			root.New.exec, root.Add.exec = exec, exec
			parser, err := newParser(&root, "v-test", deps, kong.Exit(func(int) {}))
			if err != nil {
				t.Fatal(err)
			}
			args := []string{command, "--help"}
			if command == "root" {
				args = []string{"--help"}
			}
			if _, err := parser.Parse(args); err != nil && stdout.Len() == 0 {
				t.Fatal(err)
			}
			want, err := os.ReadFile("testdata/help/" + command + ".txt")
			if err != nil {
				t.Fatal(err)
			}
			if got := stdout.String(); got != string(want) {
				t.Fatalf("%s help drifted\n--- got ---\n%s\n--- want ---\n%s", command, got, want)
			}
		})
	}
}

func TestCreateOutputIncludesChecksAndOrderedNextSteps(t *testing.T) {
	t.Parallel()
	wantSteps := []string{"cd 'demo'", "go run . greet --name World", "go build -o demo ."}
	out := createOutputFrom(generator.CreateResult{
		Checks:    []generator.CheckResult{{Command: "go test ./...", Status: "passed"}},
		NextSteps: wantSteps,
	})
	if len(out.Checks) != 1 || !reflect.DeepEqual(out.NextSteps, wantSteps) {
		t.Fatalf("output=%+v", out)
	}
}

func TestAddOutputIncludesChecks(t *testing.T) {
	t.Parallel()
	out := addOutputFrom(generator.AddResult{Checks: []generator.CheckResult{{Command: "go test ./...", Status: "passed"}}})
	if len(out.Checks) != 1 || out.Checks[0].Command != "go test ./..." {
		t.Fatalf("output=%+v", out)
	}
}

func TestExecuteDependencyPreparationFailureUsesPartialExitSeven(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) {
			return generator.CreateResult{ProjectName: "demo", Checks: []generator.CheckResult{{Command: "go get x", Status: "failed"}}},
				&generator.OperationError{Kind: generator.ErrorScaffoldFailed, Partial: true, Err: errors.New("recover with: go get x")}
		},
		add: func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"new", "demo", "--no-verify", "--json"}, "v-test", testDependencies(&stdout, &stderr, projects))
	if code != 7 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	var out createOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Outcome != "partial_success" || out.ErrorCode != generator.ErrorScaffoldFailed || len(out.Checks) != 1 {
		t.Fatalf("output=%+v", out)
	}
}

func TestRenderCreateConciseHappyPath(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	renderCreate(&output, generator.CreateResult{
		ProjectName: "tvguide", Mode: "flat", VerifyRan: true, VerifyPassed: true,
		NextSteps: []string{"cd 'tvguide'", "go run . greet --name World", "go build -o tvguide ."},
	})
	want := "Created tvguide (flat)\nVerified: tests and build\n\nNext:\n  cd 'tvguide'\n  go run . greet --name World\n  go build -o tvguide .\n"
	if output.String() != want {
		t.Fatalf("output=%q want=%q", output.String(), want)
	}
}
