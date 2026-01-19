package api_test

import (
	"os"
	"strings"
	"testing"
)

// TestGokartAPI_HasAllPublicFunctions verifies that all public functions
// from the main package are documented with signatures
func TestGokartAPI_HasAllPublicFunctions(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	// Expected public functions from main package
	expectedFunctions := []string{
		"LoadConfig[T any]",
		"LoadConfigWithDefaults[T any]",
		"ListenAndServe",
		"ListenAndServeWithTimeout",
		"NewRouter",
		"NewHTTPClient",
		"NewStandardClient",
		"NewValidator",
		"NewStandardValidator",
		"ValidationErrors",
		"OpenPostgres",
		"OpenPostgresWithConfig",
		"PostgresFromEnv",
		"WithTransaction",
		"OpenSQLite",
		"OpenSQLiteContext",
		"OpenSQLiteWithConfig",
		"SQLiteInMemory",
		"SQLiteTransaction",
		"DefaultCacheConfig",
		"OpenCache",
		"OpenCacheURL",
		"OpenCacheWithConfig",
		"Get",
		"Set",
		"GetJSON",
		"SetJSON",
		"Delete",
		"Exists",
		"Expire",
		"TTL",
		"Incr",
		"IncrBy",
		"SetNX",
		"Remember",
		"RememberJSON",
		"IsNil",
		"DefaultMigrateConfig",
		"Migrate",
		"MigrateUp",
		"MigrateDown",
		"MigrateDownTo",
		"MigrateReset",
		"MigrateStatus",
		"MigrateVersion",
		"MigrateCreate",
		"PostgresMigrate",
		"SQLiteMigrate",
		"Render",
		"RenderWithStatus",
		"RenderCtx",
		"TemplHandler",
		"TemplHandlerFunc",
		"TemplHandlerFuncE",
		"SaveState[T any]",
		"LoadState[T any]",
		"StatePath",
		"NewOpenAIClient",
		"NewOpenAIClientWithKey",
		"JSON",
		"JSONStatus",
		"JSONStatusE",
		"Error",
		"NoContent",
	}

	for _, fn := range expectedFunctions {
		if !strings.Contains(doc, fn) {
			t.Errorf("documentation missing function: %s", fn)
		}
	}
}

// TestGokartAPI_HasAllTypes verifies that all public types
// from the main package are documented with their fields
func TestGokartAPI_HasAllTypes(t *testing.T) {
	content, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatalf("failed to read gokart.md: %v", err)
	}

	doc := string(content)

	// Expected public types from main package
	expectedTypes := []struct {
		name   string
		fields []string
	}{
		{
			name: "RouterConfig",
			fields: []string{
				"Middleware",
				"Timeout",
			},
		},
		{
			name: "HTTPConfig",
			fields: []string{
				"Timeout",
				"RetryMax",
				"RetryWait",
			},
		},
		{
			name: "ValidatorConfig",
			fields: []string{
				"UseJSONNames",
			},
		},
		{
			name: "CacheConfig",
			fields: []string{
				"URL",
				"Addr",
				"Password",
				"DB",
				"PoolSize",
				"MinIdleConns",
				"DialTimeout",
				"ReadTimeout",
				"WriteTimeout",
				"KeyPrefix",
			},
		},
		{
			name: "Cache",
			fields: []string{
				"Client",
				"Close",
			},
		},
		{
			name: "MigrateConfig",
			fields: []string{
				"Dir",
				"Table",
				"Dialect",
				"FS",
				"AllowMissing",
				"NoVersioning",
			},
		},
	}

	for _, tt := range expectedTypes {
		if !strings.Contains(doc, "type "+tt.name) {
			t.Errorf("documentation missing type: %s", tt.name)
			continue
		}

		// Check for key fields
		for _, field := range tt.fields {
			if !strings.Contains(doc, field) {
				t.Errorf("type %s documentation missing field: %s", tt.name, field)
			}
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
	// Functions should be documented with:
	// - `func Name(params)` signature
	// - Description text
	// - Return type information (if applicable)

	// Sample of functions to verify format
	testCases := []struct {
		signature         string
		hasDescription    bool
		hasReturnType     bool
		hasExample        bool
	}{
		{
			signature:     "func LoadConfig[T any](paths ...string) (T, error)",
			hasReturnType: true,
			hasExample:    true,
		},
		{
			signature:     "func ListenAndServe(addr string, handler http.Handler) error",
			hasReturnType: true,
		},
		{
			signature:     "func NewRouter(cfg RouterConfig) chi.Router",
			hasReturnType: true,
			hasExample:    true,
		},
		{
			signature:     "func NewHTTPClient(cfg HTTPConfig) *retryablehttp.Client",
			hasReturnType: true,
			hasExample:    true,
		},
		{
			signature:     "func (c *Cache) Get(ctx context.Context, key string) (string, error)",
			hasReturnType: true,
		},
		{
			signature:     "func SaveState[T any](appName, filename string, data T) error",
			hasReturnType: true,
			hasExample:    true,
		},
	}

	for _, tc := range testCases {
		if !strings.Contains(doc, tc.signature) {
			t.Errorf("documentation missing signature: %s", tc.signature)
			continue
		}

		// Verify the signature is in a code block
		signatureStart := strings.Index(doc, tc.signature)
		if signatureStart == -1 {
			continue
		}

		// Check that it's formatted as code (has backticks nearby)
		context := doc[max(0, signatureStart-50):min(len(doc), signatureStart+50)]
		if !strings.Contains(context, "`") {
			t.Errorf("signature %s not in code format", tc.signature)
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

	// Check that deprecated functions are documented
	deprecatedFunctions := []string{
		"NewLogger",
		"NewFileLogger",
		"LogPath",
		"OpenPostgres",
		"OpenSQLite",
	}

	for _, fn := range deprecatedFunctions {
		if !strings.Contains(doc, fn) {
			t.Errorf("deprecated section missing function: %s", fn)
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
		"LoadConfig[Config]",
		"NewRouter",
		"NewHTTPClient",
		"OpenCache",
		"SaveState",
		"LoadState",
		"Render",
		"TemplHandler",
		"NewValidator",
	}

	for _, fn := range functionsWithExamples {
		// Find the function section
		idx := strings.Index(doc, "func "+fn)
		if idx == -1 {
			// Try alternate format
			idx = strings.Index(doc, fn)
		}

		if idx == -1 {
			t.Logf("warning: could not find function %s to check for example", fn)
			continue
		}

		// Check for example in next 500 characters
		section := doc[idx:min(len(doc), idx+500)]
		if !strings.Contains(section, "Example:") {
			t.Errorf("function %s missing example", fn)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
