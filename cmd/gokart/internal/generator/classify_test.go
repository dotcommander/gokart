package generator

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestClassifyScaffoldError(t *testing.T) {
	t.Parallel()
	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		err := classifyScaffoldError(&ExistingFileConflictError{Paths: []string{"a.txt", "b.txt"}})
		var op *OperationError
		if !errors.As(err, &op) || op.Kind != ErrorExistingFileConflict {
			t.Fatalf("classify = %v", err)
		}
		if len(op.Conflicts) != 2 || op.Conflicts[0] != "a.txt" {
			t.Errorf("conflicts = %v", op.Conflicts)
		}
	})
	t.Run("lock", func(t *testing.T) {
		t.Parallel()
		err := classifyScaffoldError(&ApplyLockError{TargetDir: "/tmp/x"})
		var op *OperationError
		if !errors.As(err, &op) || op.Kind != ErrorTargetLocked {
			t.Fatalf("classify = %v", err)
		}
	})
	t.Run("other", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("boom")
		err := classifyScaffoldError(boom)
		var op *OperationError
		if !errors.As(err, &op) || op.Kind != ErrorScaffoldFailed {
			t.Fatalf("classify = %v", err)
		}
		if !errors.Is(err, boom) {
			t.Error("must wrap original")
		}
	})
}

func TestFormatPathList(t *testing.T) {
	t.Parallel()
	if got := formatPathList([]string{"a", "b"}); got != "a, b" {
		t.Errorf("got %q", got)
	}
	many := make([]string, 13)
	for i := range many {
		many[i] = fmt.Sprintf("p%d", i)
	}
	if got := formatPathList(many); !strings.Contains(got, "and 3 more") || !strings.Contains(got, "p0") {
		t.Errorf("truncated = %q", got)
	}
}
