package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var markdownLink = regexp.MustCompile(`\[[^]]+\]\(([^)]+)\)`)

func TestRelativeMarkdownLinksResolve(t *testing.T) {
	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		return checkMarkdownFileLinks(t, path, entry, err)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkMarkdownFileLinks(t *testing.T, path string, entry os.DirEntry, walkErr error) error {
	t.Helper()
	if entry != nil && entry.IsDir() && strings.HasPrefix(entry.Name(), ".") && path != ".." {
		return filepath.SkipDir
	}
	if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".md" {
		return walkErr
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, match := range markdownLink.FindAllStringSubmatch(markdownProse(data), -1) {
		checkMarkdownLink(t, path, match[1])
	}
	return nil
}

func markdownProse(data []byte) string {
	var prose strings.Builder
	inFence := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			prose.WriteString(stripInlineCode(line))
			prose.WriteByte('\n')
		}
	}
	return prose.String()
}

func checkMarkdownLink(t *testing.T, sourcePath, target string) {
	t.Helper()
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "/") {
		return
	}
	file, anchor, _ := strings.Cut(target, "#")
	targetPath := sourcePath
	if file != "" {
		targetPath = filepath.Join(filepath.Dir(sourcePath), filepath.FromSlash(file))
		if _, err := os.Stat(targetPath); err != nil {
			t.Errorf("%s has broken link %q", sourcePath, target)
			return
		}
	}
	if anchor != "" && !markdownHasAnchor(t, targetPath, anchor) {
		t.Errorf("%s has broken anchor %q", sourcePath, target)
	}
}

func markdownHasAnchor(t *testing.T, path, want string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
		slug := strings.ToLower(heading)
		slug = regexp.MustCompile(`[^a-z0-9 -]`).ReplaceAllString(slug, "")
		slug = strings.ReplaceAll(slug, " ", "-")
		if slug == want {
			return true
		}
	}
	return false
}

func stripInlineCode(line string) string {
	var out strings.Builder
	for {
		before, rest, found := strings.Cut(line, "`")
		out.WriteString(before)
		if !found {
			return out.String()
		}
		_, line, found = strings.Cut(rest, "`")
		if !found {
			return out.String()
		}
	}
}

func TestDocsDoNotUseRetiredImportsOrUnpinnedInstalls(t *testing.T) {
	retiredImports := []string{"github.com/dotcommander/gokart/ai", "github.com/dotcommander/gokart/fs"}
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, text := range retiredImports {
			if strings.Contains(string(data), text) {
				t.Errorf("%s references retired import %s", path, text)
			}
		}
		if strings.Contains(string(data), "@latest") {
			t.Errorf("%s uses non-reproducible @latest", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemovedPublicIdentifiersHaveMigrationGuidance(t *testing.T) {
	data, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(data)
	for _, identifier := range []string{
		"ai.NewOpenAIClient", "fs.ConfigDir", "GetString", "cli.Fatal",
		"SetOutput", "HealthHandler", "TemplHandler", "RateLimit",
	} {
		if !strings.Contains(migration, identifier) {
			t.Errorf("README migration guidance omits removed identifier %s", identifier)
		}
	}
}

func TestGettingStartedCommandsMatchCLIPathRules(t *testing.T) {
	data, err := os.ReadFile("getting-started.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if strings.Contains(doc, "gokart new . ") {
		t.Fatal("getting started uses rejected project path '.'; use an absolute path or valid basename")
	}
	for _, command := range []string{
		"gokart new mycli --db sqlite --example",
		"gokart new service --structured --global",
		`gokart new "$PWD" --verify-only`,
	} {
		if !strings.Contains(doc, command) {
			t.Errorf("getting started omits verified command %q", command)
		}
	}
}
