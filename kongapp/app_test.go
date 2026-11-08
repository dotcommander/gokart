package kongapp_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/dotcommander/gokart/kongapp"
)

type testTree struct {
	Echo testEchoCmd `cmd:""`
}

type testEchoCmd struct {
	Text string `arg:""`
}

type testDependency struct {
	Suffix string
}

func (c *testEchoCmd) Run(ctx context.Context, streams kongapp.Streams, dep *testDependency) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := fmt.Fprint(streams.Out, c.Text+dep.Suffix)
	return err
}

func TestExecuteRunsLifecycleWithCallerOwnedDependencies(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	order := make([]string, 0, 3)
	tree := &testTree{}
	err := kongapp.Execute(t.Context(), []string{"echo", "hello"}, kongapp.Streams{
		Out: &stdout,
		Err: &stderr,
	}, tree, kongapp.Config[testTree]{
		Name:        "example",
		Description: "example command",
		Options:     []kong.Option{kong.Bind(&testDependency{Suffix: "!"})},
		Initialize: func(_ context.Context, parsed *kong.Context, populated *testTree, args []string) (kongapp.Cleanup, error) {
			order = append(order, "initialize")
			if parsed.Command() != "echo <text>" {
				t.Fatalf("command = %q", parsed.Command())
			}
			if populated.Echo.Text != "hello" {
				t.Fatalf("text = %q", populated.Echo.Text)
			}
			if got := fmt.Sprint(args); got != "[echo hello]" {
				t.Fatalf("args = %s", got)
			}
			return func() error {
				order = append(order, "cleanup")
				return nil
			}, nil
		},
	})
	order = append(order, "returned")

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := stdout.String(); got != "hello!" {
		t.Fatalf("stdout = %q", got)
	}
	if got := fmt.Sprint(order); got != "[initialize cleanup returned]" {
		t.Fatalf("order = %s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteMetadataSkipsInitialization(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	initialized := false
	tree := &testTree{}
	err := kongapp.Execute(t.Context(), []string{"--help"}, kongapp.Streams{
		Out: &stdout,
		Err: &stderr,
	}, tree, kongapp.Config[testTree]{
		Name: "example",
		Initialize: func(context.Context, *kong.Context, *testTree, []string) (kongapp.Cleanup, error) {
			initialized = true
			return nil, nil
		},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if initialized {
		t.Fatal("initializer ran for help")
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage:")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteUsesDefaultArgsWithoutMutatingConfig(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	defaults := []string{"echo", "default"}
	tree := &testTree{}
	err := kongapp.Execute(t.Context(), nil, kongapp.Streams{
		Out: &stdout,
		Err: &stderr,
	}, tree, kongapp.Config[testTree]{
		Name:        "example",
		DefaultArgs: defaults,
		Options:     []kong.Option{kong.Bind(&testDependency{})},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := stdout.String(); got != "default" {
		t.Fatalf("stdout = %q", got)
	}
	if got := fmt.Sprint(defaults); got != "[echo default]" {
		t.Fatalf("defaults mutated: %s", got)
	}
}

func TestExecuteJoinsCommandAndCleanupFailures(t *testing.T) {
	t.Parallel()

	commandErr := errors.New("command failed")
	cleanupErr := errors.New("cleanup failed")
	tree := &failingTree{Fail: failingCmd{Err: commandErr}}
	err := kongapp.Execute(t.Context(), []string{"fail"}, kongapp.Streams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}, tree, kongapp.Config[failingTree]{
		Name: "example",
		Initialize: func(context.Context, *kong.Context, *failingTree, []string) (kongapp.Cleanup, error) {
			return func() error { return cleanupErr }, nil
		},
	})

	if !errors.Is(err, commandErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("Execute() error = %v", err)
	}
}

type failingTree struct {
	Fail failingCmd `cmd:""`
}

type failingCmd struct {
	Err error `kong:"-"`
}

func (c *failingCmd) Run() error { return c.Err }
