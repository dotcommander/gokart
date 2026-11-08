package commands

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

func TestCollectNewWizardMapsAnswers(t *testing.T) {
	t.Parallel()
	flat := wizardAnswersToRequest(wizardAnswers{Project: "demo", Layout: layoutFlat, Example: true}, "/w", nil)
	if !flat.Flat || flat.Structured || !flat.Example || !reflect.DeepEqual(flat.Args, []string{"demo"}) || flat.WorkingDir != "/w" {
		t.Errorf("flat mapping wrong: %+v", flat)
	}
	st := wizardAnswersToRequest(wizardAnswers{Project: "demo", Layout: layoutStructured, DB: "sqlite", AI: true, Redis: true, Verify: true}, "/w", nil)
	if !st.Structured || st.Flat || st.DB != "sqlite" || !st.AI || !st.Redis || !st.Verify || !reflect.DeepEqual(st.Args, []string{"demo"}) {
		t.Errorf("structured mapping wrong: %+v", st)
	}
	if empty := wizardAnswersToRequest(wizardAnswers{}, "/w", nil); len(empty.Args) != 0 {
		t.Errorf("empty project Args = %v", empty.Args)
	}
}

func TestIsTerminal(t *testing.T) {
	t.Parallel()
	if isTerminal(&bytes.Buffer{}) || isTerminal(nil) {
		t.Fatal("non-file writers must not be terminals")
	}
}

func TestNewWithoutProjectNonInteractiveReturnsExit2(t *testing.T) {
	t.Parallel()
	called := false
	projects := fakeProjects{
		create: func(generator.CreateRequest) (generator.CreateResult, error) {
			called = true
			return generator.CreateResult{}, nil
		},
		add: func(generator.AddRequest) (generator.AddResult, error) { return generator.AddResult{}, nil },
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"new"}, "v-test", testDependencies(&stdout, &stderr, projects)); code != 2 {
		t.Fatalf("exit=%d, want 2; stderr=%q", code, stderr.String())
	}
	if called {
		t.Fatal("Create must not be called")
	}
}
