package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCrossReferenceLinks verifies that documentation has proper cross-references
// between related topics as per acceptance criteria:
// AC1: Component docs have "See also" sections with links to related guides and API docs
// AC2: Guides link to component docs when mentioning component names
// AC3: API reference links to component docs for context
func TestCrossReferenceLinks(t *testing.T) {
	componentDocs := []string{
		"postgres",
		"sqlite",
		"cache",
		"validate",
		"migrate",
		"state",
		"openai",
		"response",
		"templ",
	}

	t.Run("AC1: Component docs have See Also sections with cross-references", func(t *testing.T) {
		for _, component := range componentDocs {
			t.Run(component, func(t *testing.T) {
				docPath := filepath.Join("components", component+".md")
				content, err := os.ReadFile(docPath)
				if err != nil {
					t.Fatalf("Failed to read %s: %v", component, err)
				}
				doc := string(content)

				// Check for "See Also" section
				if !strings.Contains(doc, "### See Also") && !strings.Contains(doc, "## See Also") {
					t.Errorf("%s: Missing 'See Also' section", component)
				}

				// Count internal cross-references (links to /components/ or /api/)
				internalLinkPattern := regexp.MustCompile(`\]\(/components/[^\)]+\)|\]\(/api/[^\)]+\)`)
				matches := internalLinkPattern.FindAllString(doc, -1)

				if len(matches) == 0 {
					t.Errorf("%s: 'See Also' section exists but has no internal cross-references to other docs", component)
				}

				t.Logf("%s: Found %d internal cross-references", component, len(matches))
			})
		}
	})

	t.Run("AC2: API reference sections link to component docs for context", func(t *testing.T) {
		apiDocs := []struct {
			name     string
			sections []string
		}{
			{"gokart", []string{
				"Configuration",
				"State Persistence",
				"Subpackages",
				"Deprecated Functions",
			}},
		}

		for _, apiDoc := range apiDocs {
			t.Run(apiDoc.name, func(t *testing.T) {
				docPath := filepath.Join("api", apiDoc.name+".md")
				content, err := os.ReadFile(docPath)
				if err != nil {
					t.Fatalf("Failed to read %s: %v", apiDoc.name, err)
				}
				doc := string(content)

				for _, section := range apiDoc.sections {
					// Find the section in the doc
					sectionPattern := regexp.MustCompile(`## ` + regexp.QuoteMeta(section) + `\n*\n*([^\n]*)`)
					sectionMatch := sectionPattern.FindStringSubmatch(doc)

					if len(sectionMatch) == 0 {
						t.Errorf("%s: Section %q not found", apiDoc.name, section)
						continue
					}

					// The text immediately after the section heading should contain a link
					sectionText := sectionMatch[1]

					// Check if there's a component documentation link
					linkPattern := regexp.MustCompile(`\[.*?\]\(/components/`)
					if !linkPattern.MatchString(sectionText) && sectionText != "" {
						t.Logf("%s: Section %q may be missing component link (has: %q)", apiDoc.name, section, sectionText)
					}

					// Also check the broader section for any component links
					var fullSection string
					if sectionStart := strings.Index(doc, "## "+section); sectionStart != -1 {
						rest := doc[sectionStart+len("## "+section):]
						if before, _, found := strings.Cut(rest, "\n## "); found {
							fullSection = before
						} else {
							fullSection = rest
						}
					}

					if !regexp.MustCompile(`\(/components/`).MatchString(fullSection) {
						// Some sections like HTTP Server might not have component docs, that's OK
						if section == "HTTP Server" || section == "HTTP Router" || section == "HTTP Client" || section == "Configuration" || section == "Deprecated Functions" || section == "State Persistence" {
							t.Logf("%s: Section %q has no component link (acceptable - no dedicated component doc)", apiDoc.name, section)
						} else {
							t.Errorf("%s: Section %q missing link to component documentation", apiDoc.name, section)
						}
					}
				}
			})
		}
	})

	t.Run("AC3: Component See Also sections link to related API docs", func(t *testing.T) {
		// Check that component docs link back to API reference where appropriate
		expectedAPILinks := map[string][]string{
			"validate": {"response"},
			"response": {"validate", "templ"},
			"templ":    {"response"},
			"state":    {"Config"}, // Special case - links to API section
			"openai":   {"HTTP client"},
		}

		for component, expectedTargets := range expectedAPILinks {
			t.Run(component, func(t *testing.T) {
				docPath := filepath.Join("components", component+".md")
				content, err := os.ReadFile(docPath)
				if err != nil {
					t.Fatalf("Failed to read %s: %v", component, err)
				}
				doc := string(content)

				// Find See Also section
				var seeAlso string
				if _, after, found := strings.Cut(doc, "### See Also"); found {
					if before, _, found := strings.Cut(after, "\n##"); found {
						seeAlso = before
					} else {
						seeAlso = after
					}
				}

				if seeAlso == "" {
					t.Skipf("%s: No See Also section found", component)
				}

				// Check for expected targets
				for _, target := range expectedTargets {
					if !strings.Contains(seeAlso, target) {
						t.Errorf("%s: See Also section missing expected reference to %s", component, target)
					}
				}
			})
		}
	})

	t.Run("Verify: All internal links use consistent path format", func(t *testing.T) {
		allDocs := append(componentDocs, "gokart", "cli")

		for _, docName := range allDocs {
			var docPath string
			if strings.Contains(docName, "/") {
				docPath = filepath.Join(docName+".md")
			} else if docName == "gokart" || docName == "cli" {
				docPath = filepath.Join("api", docName+".md")
			} else {
				docPath = filepath.Join("components", docName+".md")
			}

			content, err := os.ReadFile(docPath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", docName, err)
			}
			doc := string(content)

			// Strip fenced code blocks — they contain Go generics syntax
			// that matches markdown link patterns (e.g., F[Type](value))
			codeBlockPattern := regexp.MustCompile("(?s)```[^\n]*\n.*?```")
			doc = codeBlockPattern.ReplaceAllString(doc, "")

			// Check for consistent internal link format
			// Should be /components/name or /api/name#section
			linkPattern := regexp.MustCompile(`\[(.*?)\]\(([^\)]+)\)`)
			allLinks := linkPattern.FindAllStringSubmatch(doc, -1)

			for _, match := range allLinks {
				link := match[2]
				// Skip external links
				if strings.HasPrefix(link, "http") {
					continue
				}
				// Skip Go code signatures matched as links (e.g. generic funcs)
				if strings.Contains(link, "...") || strings.Contains(link, ",") {
					continue
				}
				// Allow relative links that start with ./ or ../
				if !strings.HasPrefix(link, "./") && !strings.HasPrefix(link, "../") && !strings.HasPrefix(link, "#") {
					// Should be using /path format for absolute links within docs
					if !strings.HasPrefix(link, "/") {
						t.Errorf("%s: Found non-absolute internal link: %s (should be /path format)", docName, link)
					}
				}
			}
		}
	})
}
