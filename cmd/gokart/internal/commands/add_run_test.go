package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

func addDeps(stdout, stderr *bytes.Buffer, projects ProjectGenerator) Dependencies {
	return testDependencies(stdout, stderr, projects)
}

func TestAddJSONSuccess(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add: func(req generator.AddRequest) (generator.AddResult, error) {
			return generator.AddResult{Added: []string{"sqlite"}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"add", "sqlite", "--json"}, "v-test", addDeps(&stdout, &stderr, projects))
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	var out addOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Outcome != "success" || len(out.Added) != 1 {
		t.Fatalf("out=%+v", out)
	}
}

func TestAddHumanOutput(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add: func(generator.AddRequest) (generator.AddResult, error) {
			return generator.AddResult{Added: []string{"redis"}, AlreadyPresent: []string{"sqlite"}}, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"add", "redis"}, "v-test", addDeps(&stdout, &stderr, projects))
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Added redis")) || !bytes.Contains(stdout.Bytes(), []byte("Warning: sqlite already enabled")) {
		t.Errorf("stdout = %q", stdout.String())
	}
}

func TestAddJSONErrorClassified(t *testing.T) {
	t.Parallel()
	boom := errors.New("no manifest")
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, boom },
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"add", "sqlite", "--json"}, "v-test", addDeps(&stdout, &stderr, projects))
	if code != 1 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var out addOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Outcome != "failure" {
		t.Fatalf("out=%+v", out)
	}
}

func TestAddGetwdError(t *testing.T) {
	t.Parallel()
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) { return generator.CreateResult{}, nil },
		add:    func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	var stdout, stderr bytes.Buffer
	deps := testDependencies(&stdout, &stderr, projects)
	deps.Getwd = func() (string, error) { return "", errors.New("no cwd") }
	code := Execute(context.Background(), []string{"add", "sqlite"}, "v-test", deps)
	if code == 0 {
		t.Fatal("exit must be non-zero when Getwd fails")
	}
}
