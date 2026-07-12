package components_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResponseDocs_AcceptanceCriteria(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "components", "response.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("Failed to read response.md: %v", err)
	}
	doc := string(content)

	t.Run("AC1: All response helper functions are listed", func(t *testing.T) {
		verifyResponseFunctionsListed(t, doc)
	})
	t.Run("AC2: Error response format is documented", func(t *testing.T) {
		runDocContentChecks(t, doc, errorFormatChecks())
	})
	t.Run("AC3: Handler using helpers is shown", func(t *testing.T) {
		runDocContentChecks(t, doc, handlerExampleChecks())
	})
	t.Run("Additional verification: Reference section structure", func(t *testing.T) {
		verifyReferenceSectionStructure(t, doc)
	})
}

// docCheck is one named bundle of required substrings that must appear in a doc.
type docCheck struct {
	name     string
	required []string
}

// runDocContentChecks runs each docCheck as a t.Run subtest that asserts every
// required substring is present in doc.
func runDocContentChecks(t *testing.T, doc string, checks []docCheck) {
	t.Helper()
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			for _, req := range check.required {
				if !strings.Contains(doc, req) {
					t.Errorf("Documentation missing required content: %q\nIn section: %s", req, check.name)
				}
			}
		})
	}
}

func verifyResponseFunctionsListed(t *testing.T, doc string) {
	t.Helper()

	requiredFunctions := []string{
		"func JSON(w http.ResponseWriter, data any)",
		"func JSONStatus(w http.ResponseWriter, status int, data any)",
		"func JSONStatusE(w http.ResponseWriter, status int, data any) error",
		"func Error(w http.ResponseWriter, status int, message string)",
		"func NoContent(w http.ResponseWriter)",
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(doc, fn) {
			t.Errorf("Documentation missing function signature: %q", fn)
		}
	}

	referenceRows := []struct {
		marker  string
		errText string
	}{
		{"| `JSON`", "Reference section missing JSON function"},
		{"| `JSONStatus`", "Reference section missing JSONStatus function"},
		{"| `JSONStatusE`", "Reference section missing JSONStatusE function"},
		{"| `Error`", "Reference section missing Error function"},
		{"| `NoContent`", "Reference section missing NoContent function"},
	}
	for _, row := range referenceRows {
		if !strings.Contains(doc, row.marker) {
			t.Error(row.errText)
		}
	}
}

func errorFormatChecks() []docCheck {
	return []docCheck{
		{
			name: "Error function signature",
			required: []string{
				"func Error(w http.ResponseWriter, status int, message string)",
			},
		},
		{
			name: "Error response format example",
			required: []string{
				`"error": "Error message here"`,
			},
		},
		{
			name: "Common error status codes",
			required: []string{
				"- `400 Bad Request`",
				"- `401 Unauthorized`",
				"- `403 Forbidden`",
				"- `404 Not Found`",
				"- `409 Conflict`",
				"- `422 Unprocessable Entity`",
				"- `500 Internal Server Error`",
			},
		},
		{
			name: "Error response example with 400",
			required: []string{
				`web.Error(w, http.StatusBadRequest, "Missing user ID")`,
				`// Response: 400 {"error":"Missing user ID"}`,
			},
		},
	}
}

func handlerExampleChecks() []docCheck {
	return []docCheck{
		{
			name: "Quick start example handler",
			required: []string{
				"func handleUser(w http.ResponseWriter, r *http.Request)",
				`web.JSON(w, user)`,
				`web.Error(w, http.StatusNotFound, "user not found")`,
				"return",
			},
		},
		{
			name: "REST API handler example",
			required: []string{
				"func handleUser(w http.ResponseWriter, r *http.Request)",
				"id := r.PathValue(\"id\")",
				`web.Error(w, http.StatusNotFound, "User not found")`,
				`web.Error(w, http.StatusInternalServerError, "Failed to fetch user")`,
				`web.JSON(w, user)`,
			},
		},
		{
			name: "Create handler example",
			required: []string{
				"func handleCreate(w http.ResponseWriter, r *http.Request)",
				"var req CreateUserRequest",
				`web.Error(w, http.StatusBadRequest, "Invalid request body")`,
				`web.Error(w, http.StatusUnprocessableEntity, "Validation failed")`,
				`web.JSONStatus(w, http.StatusCreated, user)`,
			},
		},
		{
			name: "Delete handler example with NoContent",
			required: []string{
				"func handleDelete(w http.ResponseWriter, r *http.Request)",
				`web.Error(w, http.StatusInternalServerError, "Failed to delete user")`,
				"web.NoContent(w)",
				"// Response: 204 (no body)",
			},
		},
		{
			name: "JSON function example",
			required: []string{
				"func handleUser(w http.ResponseWriter, r *http.Request)",
				`user := User{ID: 1, Name: "Alice"}`,
				`web.JSON(w, user)`,
				`// Response: 200 {"id":1,"name":"Alice"}`,
			},
		},
	}
}

func verifyReferenceSectionStructure(t *testing.T, doc string) {
	t.Helper()

	required := []string{
		"## Reference",
		"### Functions",
		"| Function | Returns | Description |",
		"| `JSON` | - | Write 200 JSON response |",
		"| `JSONStatus` | - | Write JSON response with custom status |",
		"| `JSONStatusE` | `error` | Write JSON response, return error on failure |",
		"| `Error` | - | Write JSON error response |",
		"| `NoContent` | - | Write 204 No Content response |",
	}
	for _, req := range required {
		if !strings.Contains(doc, req) {
			t.Errorf("Documentation missing required content: %q", req)
		}
	}
}
