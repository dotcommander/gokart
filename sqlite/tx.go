package sqlite

import (
	"errors"
	"fmt"
)

// runTx executes fn within a transaction lifecycle supplied by the caller.
// It preserves rollback-on-error/panic and commit-on-success semantics.
//
// This helper is intentionally module-local: the sqlite module must not
// depend on the root gokart module (see PHILOSOPHY.md, dependency
// isolation). The postgres module carries its own copy.
func runTx[T any](begin func() (T, error), rollback func(T) error, commit func(T) error, fn func(T) error) (err error) {
	tx, err := begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = rollback(tx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := rollback(tx); rbErr != nil {
			return errors.Join(fmt.Errorf("rollback failed: %w", rbErr), err)
		}
		return err
	}

	if err := commit(tx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
