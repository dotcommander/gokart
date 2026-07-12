package postgres

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestNewPostgresIdentifier(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"memory_memories", "embedding_cache", "MixedCase", "Select"} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := NewPostgresIdentifier(input)
			if err != nil {
				t.Fatalf("NewPostgresIdentifier() error = %v", err)
			}
			if got.Raw != input || got.Quoted != (pgx.Identifier{input}).Sanitize() {
				t.Fatalf("NewPostgresIdentifier() = %#v", got)
			}
		})
	}
}

func TestNewPostgresIdentifierRejectsInvalidNames(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", " ", "table;DROP", `bad"name`, "public.table", "has space", "has-hyphen", "bad\x00name", "1leading_digit", "naiveé", strings.Repeat("a", 64)} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if _, err := NewPostgresIdentifier(input); err == nil {
				t.Fatalf("NewPostgresIdentifier(%q) succeeded, want error", input)
			}
		})
	}
}

func TestNewPostgresIndexIdentifier(t *testing.T) {
	t.Parallel()

	short, err := NewPostgresIndexIdentifier("memory_memories", "user_id")
	if err != nil {
		t.Fatalf("short identifier: %v", err)
	}
	if short.Raw != "idx_memory_memories_user_id" || short.Quoted != `"idx_memory_memories_user_id"` {
		t.Fatalf("short identifier = %#v", short)
	}

	table := strings.Repeat("a", 63)
	suffix := strings.Repeat("b", 40)
	first, err := NewPostgresIndexIdentifier(table, suffix)
	if err != nil {
		t.Fatalf("long identifier: %v", err)
	}
	second, err := NewPostgresIndexIdentifier(table, suffix)
	if err != nil {
		t.Fatalf("repeated long identifier: %v", err)
	}
	if first != second || len(first.Raw) > postgresIdentifierMaxBytes || !strings.HasPrefix(first.Raw, "idx_") {
		t.Fatalf("long identifier = %#v, repeated = %#v", first, second)
	}
}

func TestNewPostgresIndexIdentifierRejectsInvalidParts(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ table, suffix string }{
		{table: "bad.table", suffix: "index"},
		{table: "table", suffix: "bad-index"},
		{table: "table", suffix: ""},
	} {
		if _, err := NewPostgresIndexIdentifier(test.table, test.suffix); err == nil {
			t.Fatalf("NewPostgresIndexIdentifier(%q, %q) succeeded", test.table, test.suffix)
		}
	}
}
