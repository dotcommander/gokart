package components_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostgresDocs_AcceptanceCriteria(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "components", "postgres.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("Failed to read postgres.md: %v", err)
	}
	doc := string(content)

	t.Run("AC1: NewPostgresPool signature and config struct documented", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "Open function signature",
				required: []string{
					"postgres.Open(",
					"context.Context",
					"string",
					"*pgxpool.Pool",
				},
			},
			{
				name: "OpenWithConfig function signature",
				required: []string{
					"postgres.OpenWithConfig(",
					"postgres.Config",
				},
			},
			{
				name: "Config struct fields documented",
				required: []string{
					"URL",
					"MaxConns",
					"MinConns",
					"MaxConnLifetime",
					"MaxConnIdleTime",
					"HealthCheckPeriod",
				},
			},
			{
				name: "Config struct field types",
				required: []string{
					"`string`",
					"`int32`",
					"`time.Duration`",
				},
			},
			{
				name: "Config struct default values",
				required: []string{
					"`25`",
					"`5`",
					"`1 hour`",
					"`30 minutes`",
					"`1 minute`",
				},
			},
		}

		for _, check := range checks {
			t.Run(check.name, func(t *testing.T) {
				for _, req := range check.required {
					if !strings.Contains(doc, req) {
						t.Errorf("Documentation missing required content: %q\nIn section: %s", req, check.name)
					}
				}
			})
		}
	})

	t.Run("AC2: Transaction helper with rollback behavior explained", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "Transaction function signature",
				required: []string{
					"postgres.Transaction(",
					"func(tx pgx.Tx) error",
				},
			},
			{
				name: "Automatic commit on nil return",
				required: []string{
					"return nil",
					"commit",
				},
			},
			{
				name: "Automatic rollback on error",
				required: []string{
					"rollback",
					"error returned",
				},
			},
			{
				name: "Panic recovery behavior",
				required: []string{
					"panic",
					"caught",
				},
			},
			{
				name: "Transaction example",
				required: []string{
					"postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {",
					"tx.Exec(ctx",
				},
			},
		}

		for _, check := range checks {
			t.Run(check.name, func(t *testing.T) {
				for _, req := range check.required {
					if !strings.Contains(doc, req) {
						t.Errorf("Documentation missing required content: %q\nIn section: %s", req, check.name)
					}
				}
			})
		}
	})

	t.Run("AC3: Connection string format and env var patterns shown", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "DSN format template",
				required: []string{
					"postgres://user:password@host:port/database",
				},
			},
			{
				name: "DSN component breakdown",
				required: []string{
					"user",
					"password",
					"host",
					"port",
					"database",
					"sslmode",
				},
			},
			{
				name: "Common DSN examples",
				required: []string{
					"postgres://localhost:5432/myapp",
					"postgres://app:pass@db.example.com:5432/myapp",
					"?sslmode=",
				},
			},
			{
				name: "Environment variable patterns",
				required: []string{
					"DATABASE_URL",
					"DB_HOST",
					"DB_PORT",
					"DB_USER",
					"DB_PASSWORD",
					"DB_NAME",
				},
			},
			{
				name: "FromEnv function",
				required: []string{
					"postgres.FromEnv(",
					"DATABASE_URL",
				},
			},
		}

		for _, check := range checks {
			t.Run(check.name, func(t *testing.T) {
				for _, req := range check.required {
					if !strings.Contains(doc, req) {
						t.Errorf("Documentation missing required content: %q\nIn section: %s", req, check.name)
					}
				}
			})
		}
	})
}
