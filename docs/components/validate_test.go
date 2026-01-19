package components_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatorDocs_AcceptanceCriteria(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "components", "validate.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("Failed to read validate.md: %v", err)
	}
	doc := string(content)

	t.Run("AC1: NewValidator signature is documented", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "NewValidator function signature",
				required: []string{
					"func NewValidator(cfg ValidatorConfig) *validator.Validate",
				},
			},
			{
				name: "ValidatorConfig parameter documentation",
				required: []string{
					"| `cfg`     | `ValidatorConfig` | `{UseJSONNames: true}`  | Validator configuration |",
				},
			},
			{
				name: "NewStandardValidator function signature",
				required: []string{
					"func NewStandardValidator() *validator.Validate",
				},
			},
			{
				name: "ValidatorConfig struct documentation",
				required: []string{
					"| Field          | Type  | Default | Description                                             |",
					"| `UseJSONNames` | `bool` | `true`  | Use `json` tag names in errors instead of struct fields |",
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

	t.Run("AC2: Common validation tags are listed with examples", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "Common validation tags table",
				required: []string{
					"| Tag | Description | Example |",
					"| `required` | Field must be non-zero | `validate:\"required\"` |",
					"| `email` | Valid email format | `validate:\"email\"` |",
					"| `min` | Minimum length (strings/arrays) or value (numbers) | `validate:\"min=3\"` |",
					"| `max` | Maximum length (strings/arrays) or value (numbers) | `validate:\"max=100\"` |",
					"| `gte` | Greater than or equal to | `validate:\"gte=0\"` |",
					"| `lte` | Less than or equal to | `validate:\"lte=130\"` |",
					"| `oneof` | Value must be one of the listed options | `validate:\"oneof=red green blue\"` |",
				},
			},
			{
				name: "Custom notblank validator",
				required: []string{
					"### Custom Validator: `notblank`",
					"rejects whitespace-only",
					"validate:\"notblank\"",
				},
			},
			{
				name: "Tag comparison table",
				required: []string{
					"| Validator | Empty String | Whitespace Only | Non-Empty |",
					"| `required` | ❌ Fails     | ✅ Passes       | ✅ Passes  |",
					"| `notblank` | ❌ Fails     | ❌ Fails        | ✅ Passes  |",
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

	t.Run("AC3: Error message extraction pattern is shown", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "ValidationErrors function signature",
				required: []string{
					"func ValidationErrors(err error) map[string]string",
				},
			},
			{
				name: "Error extraction example",
				required: []string{
					"for field, msg := range gokart.ValidationErrors(err)",
					"fmt.Printf(\"%s %s\\n\", field, msg)",
				},
			},
			{
				name: "Error extraction example with map iteration",
				required: []string{
					"errors := gokart.ValidationErrors(err)",
					"for field, message := range errors",
					"fmt.Printf(\"%s: %s\\n\", field, message)",
				},
			},
			{
				name: "Return value documentation",
				required: []string{
					"- `map[string]string` on validation errors",
					"- `nil` if `err` is not a `validator.ValidationErrors`",
				},
			},
			{
				name: "Default error messages table",
				required: []string{
					"| Tag       | Message                                            |",
					"| `required` | \"is required\"                                      |",
					"| `notblank` | \"cannot be blank\"                                  |",
					"| `email`    | \"must be a valid email\"                            |",
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

	t.Run("AC4: Additional verification examples exist", func(t *testing.T) {
		checks := []struct {
			name     string
			required []string
		}{
			{
				name: "HTTP integration example",
				required: []string{
					"func handleCreateUser(w http.ResponseWriter, r *http.Request)",
					"w.WriteHeader(http.StatusUnprocessableEntity)",
					"\"error\": \"validation failed\"",
				},
			},
			{
				name: "Reference section",
				required: []string{
					"### Functions",
					"| Function                | Returns             | Description               |",
					"| `NewValidator`          | `*validator.Validate` | Creates configured validator |",
					"| `ValidationErrors`      | `map[string]string` | Extracts field errors     |",
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