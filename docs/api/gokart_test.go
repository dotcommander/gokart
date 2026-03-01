package api_test

import (
	"os"
	"strings"
	"testing"
)

// TestGokartAPI_HasAllPublicFunctions verifies that all public functions
// from the root gokart package are documented with signatures
func TestGokartAPI_HasAllPublicFunctions(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	// Expected public functions from root package (after subpackage decomposition)
	expectedFunctions := []string{
		"LoadConfig[T any]",
		"LoadConfigWithDefaults[T any]",
		"SaveState[T any]",
		"LoadState[T any]",
		"StatePath",
	}

	for _, fn := range expectedFunctions {
		if !strings.Contains(doc, fn) {
			t.Errorf("documentation missing function: %s", fn)
		}
	}
}

// TestGokartAPI_DeprecatedSection verifies that deprecated functions
// are documented in a separate section
func TestGokartAPI_DeprecatedSection(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	// Should have a deprecated section
	if !strings.Contains(doc, "## Deprecated Functions") {
		t.Error("documentation missing deprecated section")
	}

	// Check that deprecated logger functions are documented
	deprecatedFunctions := []string{
		"NewLogger",
		"NewFileLogger",
		"LogPath",
	}

	for _, fn := range deprecatedFunctions {
		if !strings.Contains(doc, fn) {
			t.Errorf("deprecated section missing function: %s", fn)
		}
	}
}

// TestGokartAPI_SubpackagesSection verifies that the doc has a
// subpackages reference section listing all moved components
func TestGokartAPI_SubpackagesSection(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	if !strings.Contains(doc, "## Subpackages") {
		t.Error("documentation missing Subpackages section")
	}

	// Check subpackage imports are listed
	subpackages := []string{
		"gokart/web",
		"gokart/postgres",
		"gokart/sqlite",
		"gokart/cache",
		"gokart/migrate",
		"gokart/ai",
		"gokart/logger",
		"gokart/cli",
	}

	for _, pkg := range subpackages {
		if !strings.Contains(doc, pkg) {
			t.Errorf("subpackages section missing package: %s", pkg)
		}
	}
}

// TestGokartAPI_Format verifies that each function has signature,
// brief description, and return type
func TestGokartAPI_Format(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	// Check for function documentation format patterns
	testCases := []struct {
		signature string
	}{
		{"func LoadConfig[T any](paths ...string) (T, error)"},
		{"func SaveState[T any](appName, filename string, data T) error"},
	}

	for _, tc := range testCases {
		if !strings.Contains(doc, tc.signature) {
			t.Errorf("documentation missing signature: %s", tc.signature)
		}
	}
}

// TestGokartAPI_HasExamples verifies that key functions have examples
func TestGokartAPI_HasExamples(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	// Functions that should have examples
	functionsWithExamples := []string{
		"LoadConfig",
		"SaveState",
		"LoadState",
	}

	for _, fn := range functionsWithExamples {
		idx := strings.Index(doc, fn)
		if idx == -1 {
			t.Logf("warning: could not find function %s to check for example", fn)
			continue
		}

		section := doc[idx:min(len(doc), idx+500)]
		if !strings.Contains(section, "Example") && !strings.Contains(section, "```go") {
			t.Errorf("function %s missing example", fn)
		}
	}
}
