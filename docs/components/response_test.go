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

		// Verify reference section lists all functions
		if !strings.Contains(doc, "| `JSON`") {
			t.Error("Reference section missing JSON function")
		}
		if !strings.Contains(doc, "| `JSONStatus`") {
			t.Error("Reference section missing JSONStatus function")
		}
		if !strings.Contains(doc, "| `JSONStatusE`") {
			t.Error("Reference section missing JSONStatusE function")
		}
		if !strings.Contains(doc, "| `Error`") {
			t.Error("Reference section missing Error function")
		}
		if !strings.Contains(doc, "| `NoContent`") {
			t.Error("Reference section missing NoContent function")
		}
	})

	t.Run("AC2: Error response format is documented", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
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
					`gokart.Error(w, http.StatusBadRequest, "Missing user ID")`,
					`// Response: 400 {"error":"Missing user ID"}`,
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

	t.Run("AC3: Handler using helpers is shown", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "Quick start example handler",
				required: []string{
					"func handleUser(w http.ResponseWriter, r *http.Request)",
					`gokart.JSON(w, user)`,
					`gokart.Error(w, http.StatusBadRequest, "Invalid user ID")`,
					`gokart.JSONStatus(w, http.StatusCreated, user)`,
				},
			},
			{
				name: "REST API handler example",
				required: []string{
					"func handleUser(w http.ResponseWriter, r *http.Request)",
					"id := r.PathValue(\"id\")",
					`gokart.Error(w, http.StatusNotFound, "User not found")`,
					`gokart.Error(w, http.StatusInternalServerError, "Failed to fetch user")`,
					`gokart.JSON(w, user)`,
				},
			},
			{
				name: "Create handler example",
				required: []string{
					"func handleCreate(w http.ResponseWriter, r *http.Request)",
					"var req CreateUserRequest",
					`gokart.Error(w, http.StatusBadRequest, "Invalid request body")`,
					`gokart.Error(w, http.StatusUnprocessableEntity, "Validation failed")`,
					`gokart.JSONStatus(w, http.StatusCreated, user)`,
				},
			},
			{
				name: "Delete handler example with NoContent",
				required: []string{
					"func handleDelete(w http.ResponseWriter, r *http.Request)",
					`gokart.Error(w, http.StatusInternalServerError, "Failed to delete user")`,
					"gokart.NoContent(w)",
					"// Response: 204 (no body)",
				},
			},
			{
				name: "JSON function example",
				required: []string{
					"func handleUser(w http.ResponseWriter, r *http.Request)",
					`user := User{ID: 1, Name: "Alice"}`,
					`gokart.JSON(w, user)`,
					`// Response: 200 {"id":1,"name":"Alice"}`,
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

	t.Run("Additional verification: Reference section structure", func(t *testing.T) {
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
	})
}
