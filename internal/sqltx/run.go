package sqltx

import "fmt"

// Run executes fn within a transaction lifecycle supplied by the caller.
// It preserves rollback-on-error/panic and commit-on-success semantics.
func Run[T any](begin func() (T, error), rollback func(T) error, commit func(T) error, fn func(T) error) (err error) {
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
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := commit(tx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
