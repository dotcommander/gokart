package components_test

import (
	"os"
	"strings"
	"testing"
)

func TestSQLiteDocumentation_NewSQLiteDB_SignatureDocumented(t *testing.T) {
	// Given: docs/components/sqlite.md exists
	content, err := os.ReadFile("docs/components/sqlite.md")
	if err != nil {
		t.Fatalf("Failed to read sqlite.md: %v", err)
	}

	doc := string(content)

	// When: I search for Open function signature
	// Then: Open function is documented
	if !strings.Contains(doc, "`sqlite.Open`") && !strings.Contains(doc, "sqlite.Open") {
		t.Error("sqlite.Open function not documented")
	}

	// And: OpenContext function is documented
	if !strings.Contains(doc, "OpenContext") {
		t.Error("OpenContext function not documented")
	}

	// And: OpenWithConfig function is documented
	if !strings.Contains(doc, "OpenWithConfig") {
		t.Error("OpenWithConfig function not documented")
	}

	// And: InMemory function is documented
	if !strings.Contains(doc, "InMemory") {
		t.Error("InMemory function not documented")
	}

	// And: Config struct is documented with options
	if !strings.Contains(doc, "Config Struct") && !strings.Contains(doc, "Config struct") {
		t.Error("Config struct not documented")
	}

	// And: Config options table exists
	configTableMarkers := []string{
		"Path", "WALMode", "BusyTimeout", "MaxOpenConns", "MaxIdleConns", "ConnMaxLifetime", "ForeignKeys",
	}
	for _, marker := range configTableMarkers {
		if !strings.Contains(doc, marker) {
			t.Errorf("Config option %q not documented in table", marker)
		}
	}
}

func TestSQLiteDocumentation_TransactionHelperExplained(t *testing.T) {
	// Given: docs/components/sqlite.md exists
	content, err := os.ReadFile("docs/components/sqlite.md")
	if err != nil {
		t.Fatalf("Failed to read sqlite.md: %v", err)
	}

	doc := string(content)

	// When: I search for transaction
	// Then: Transaction helper function is documented
	if !strings.Contains(doc, "Transaction Helper") && !strings.Contains(doc, "`sqlite.Transaction`") {
		t.Error("Transaction helper section not found")
	}

	// And: Usage example is provided
	if !strings.Contains(doc, "sqlite.Transaction(ctx, db, func(tx *sql.Tx) error") &&
		!strings.Contains(doc, "sqlite.Transaction(ctx, db") {
		t.Error("Transaction usage example not provided")
	}

	// And: Rollback behavior is explained
	if !strings.Contains(doc, "Automatic rollback") {
		t.Error("Rollback behavior not explained")
	}

	// And: Panic recovery is mentioned
	if !strings.Contains(doc, "Panic") && !strings.Contains(doc, "panic") {
		t.Error("Panic recovery not documented")
	}

	// And: Manual transaction example exists
	if !strings.Contains(doc, "Manual Transactions") {
		t.Error("Manual transaction section not found")
	}
}

func TestSQLiteDocumentation_ZeroCGOProminentlyMentioned(t *testing.T) {
	// Given: docs/components/sqlite.md exists
	content, err := os.ReadFile("docs/components/sqlite.md")
	if err != nil {
		t.Fatalf("Failed to read sqlite.md: %v", err)
	}

	doc := string(content)

	// When: I highlight CGO
	// Then: Zero CGO is prominently mentioned
	if !strings.Contains(doc, "zero CGO") && !strings.Contains(doc, "**zero CGO**") {
		t.Error("Zero CGO not prominently mentioned")
	}

	// And: Pure Go implementation is highlighted
	if !strings.Contains(doc, "pure Go") {
		t.Error("Pure Go implementation not mentioned")
	}

	// And: Benefit of standalone binary is explained
	if !strings.Contains(doc, "standalone binary") && !strings.Contains(doc, "cross-platform") {
		t.Error("Benefit of zero CGO (standalone binary) not explained")
	}

	// And: modernc.org/sqlite is referenced
	if !strings.Contains(doc, "modernc.org/sqlite") {
		t.Error("modernc.org/sqlite not referenced")
	}

	// And: Comparison to mattn/go-sqlite3 exists
	if !strings.Contains(doc, "mattn/go-sqlite3") {
		t.Error("Comparison to mattn/go-sqlite3 not provided")
	}
}

func TestSQLiteDocumentation_Structure(t *testing.T) {
	// Given: docs/components/sqlite.md exists
	content, err := os.ReadFile("docs/components/sqlite.md")
	if err != nil {
		t.Fatalf("Failed to read sqlite.md: %v", err)
	}

	doc := string(content)

	// Then: Follows Laravel-Rust style with proper sections
	requiredSections := []string{
		"## Installation",
		"## Quick Start",
		"## Connection",
		"## Configuration",
		"## Transactions",
		"## Querying",
		"## Best Practices",
		"## Type Mapping",
		"## Reference",
	}

	for _, section := range requiredSections {
		if !strings.Contains(doc, section) {
			t.Errorf("Required section %q not found", section)
		}
	}
}

func TestSQLiteDocumentation_CodeExamples(t *testing.T) {
	// Given: docs/components/sqlite.md exists
	content, err := os.ReadFile("docs/components/sqlite.md")
	if err != nil {
		t.Fatalf("Failed to read sqlite.md: %v", err)
	}

	doc := string(content)

	// Then: Code examples are present
	codeExamplePatterns := []string{
		"```go",
		"sqlite.Open",
		"QueryRowContext",
		"ExecContext",
	}

	for _, pattern := range codeExamplePatterns {
		if !strings.Contains(doc, pattern) {
			t.Errorf("Code example pattern %q not found", pattern)
		}
	}
}
