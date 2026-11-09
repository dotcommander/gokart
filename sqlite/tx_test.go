package sqlite

import (
	"errors"
	"strings"
	"testing"
)

type fakeTx struct{ n int }

func TestRunTxLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("begin error", func(t *testing.T) {
		t.Parallel()
		err := runTx(func() (*fakeTx, error) { return nil, errors.New("no begin") },
			func(tx *fakeTx) error { return nil },
			func(tx *fakeTx) error { return nil },
			func(tx *fakeTx) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "begin transaction") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("fn error rolls back", func(t *testing.T) {
		t.Parallel()
		rolledBack := false
		err := runTx(func() (*fakeTx, error) { return &fakeTx{}, nil },
			func(tx *fakeTx) error { rolledBack = true; return nil },
			func(tx *fakeTx) error { return nil },
			func(tx *fakeTx) error { return errors.New("inner") })
		if err == nil || err.Error() != "inner" || !rolledBack {
			t.Fatalf("err=%v rolledBack=%v", err, rolledBack)
		}
	})

	t.Run("rollback failure joins", func(t *testing.T) {
		t.Parallel()
		err := runTx(func() (*fakeTx, error) { return &fakeTx{}, nil },
			func(tx *fakeTx) error { return errors.New("rb fail") },
			func(tx *fakeTx) error { return nil },
			func(tx *fakeTx) error { return errors.New("inner") })
		if err == nil || !strings.Contains(err.Error(), "rollback failed") || !strings.Contains(err.Error(), "inner") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("commit error", func(t *testing.T) {
		t.Parallel()
		err := runTx(func() (*fakeTx, error) { return &fakeTx{}, nil },
			func(tx *fakeTx) error { return nil },
			func(tx *fakeTx) error { return errors.New("commit fail") },
			func(tx *fakeTx) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "commit transaction") {
			t.Fatalf("err = %v", err)
		}
	})
}
