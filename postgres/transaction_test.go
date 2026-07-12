package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTransactionRejectsNilPool(t *testing.T) {
	err := Transaction(context.Background(), nil, func(pgx.Tx) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "nil pool") {
		t.Fatalf("Transaction error = %v, want nil pool error", err)
	}
}

func TestTransactionRejectsNilCallback(t *testing.T) {
	err := Transaction(context.Background(), &pgxpool.Pool{}, nil)
	if err == nil || !strings.Contains(err.Error(), "nil callback") {
		t.Fatalf("Transaction error = %v, want nil callback error", err)
	}
}
