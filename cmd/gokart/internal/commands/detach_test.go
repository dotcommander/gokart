package commands

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

func runDetach(t *testing.T, detach func(generator.DetachRequest) (generator.DetachResult, error), args ...string) (int, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	deps := testDependencies(&stdout, &stderr, fakeProjects{detach: detach})
	code := Execute(context.Background(), append([]string{detachCommandName}, args...), "v-test", deps)
	return code, stdout.String()
}

func TestDetachDryRunPassesFlags(t *testing.T) {
	t.Parallel()
	var got generator.DetachRequest
	code, stdout := runDetach(t, func(req generator.DetachRequest) (generator.DetachResult, error) {
		got = req
		return generator.DetachResult{DryRun: true, FilesOverwritten: []string{"internal/app/context.go"}}, nil
	}, "--dry-run", "--force")
	if code != 0 || !got.DryRun || !got.Force || got.Dir != "/tmp/work" {
		t.Fatalf("code=%d req=%+v", code, got)
	}
	if !strings.Contains(stdout, "overwrite internal/app/context.go") {
		t.Fatalf("stdout=%q", stdout)
	}
}

func TestDetachAlreadyDetached(t *testing.T) {
	t.Parallel()
	code, stdout := runDetach(t, func(generator.DetachRequest) (generator.DetachResult, error) {
		return generator.DetachResult{AlreadyDetached: true}, nil
	})
	if code != 0 || !strings.Contains(stdout, "Already detached") {
		t.Fatalf("code=%d stdout=%q", code, stdout)
	}
}

func TestDetachConflictAndFlatExitCodes(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		kind generator.ErrorKind
		code int
	}{
		"conflict": {generator.ErrorExistingFileConflict, 3},
		"flat":     {generator.ErrorFlatModeUnsupported, 10},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			code, stdout := runDetach(t, func(generator.DetachRequest) (generator.DetachResult, error) {
				return generator.DetachResult{}, &generator.OperationError{Kind: tc.kind, Err: context.Canceled}
			}, "--json")
			if code != tc.code || !strings.Contains(stdout, string(tc.kind)) {
				t.Fatalf("code=%d stdout=%q", code, stdout)
			}
		})
	}
}
