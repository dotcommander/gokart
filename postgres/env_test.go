package postgres

import (
	"context"
	"strings"
	"testing"
)

func TestFromEnvRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := FromEnv(context.Background())
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("FromEnv with empty DATABASE_URL error = %v, want DATABASE_URL error", err)
	}
}

func TestFromEnvRejectsInvalidURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "://invalid")
	_, err := FromEnv(context.Background())
	if err == nil {
		t.Fatal("FromEnv with invalid DATABASE_URL error = nil, want error")
	}
}
