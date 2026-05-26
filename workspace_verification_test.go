package gokart_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceVerifierCoversGoWorkModules(t *testing.T) {
	t.Parallel()

	goWork := readRepoFile(t, "go.work")
	verifier := readRepoFile(t, "scripts", "verify-workspace.sh")

	if !strings.Contains(verifier, "go.work") {
		t.Fatal("workspace verifier must read go.work instead of duplicating module lists")
	}

	for _, module := range parseGoWorkModules(goWork) {
		if module == "" {
			t.Fatal("go.work contains an empty module path")
		}
	}
}

func TestWorkspaceVerifierIsolatesTestHome(t *testing.T) {
	t.Parallel()

	verifier := readRepoFile(t, "scripts", "verify-workspace.sh")
	required := []string{
		"mktemp -d",
		"gokart-verify-home",
		"gokart-verify-gocache",
		`HOME="$verify_home" GOCACHE="$verify_gocache" go test ./...`,
		`GOCACHE="$verify_gocache" go build ./...`,
	}

	for _, fragment := range required {
		if !strings.Contains(verifier, fragment) {
			t.Fatalf("workspace verifier must isolate test HOME; missing %q", fragment)
		}
	}
}

func TestIgnoredExamplesAreCompileChecked(t *testing.T) {
	t.Parallel()

	verifier := readRepoFile(t, "scripts", "verify-workspace.sh")
	examples := []string{
		"docs/examples/cache/main.go",
		"docs/examples/config/main.go",
		"docs/examples/full-service/main.go",
		"docs/examples/http-server/main.go",
		"docs/examples/logger/main.go",
		"docs/examples/openai/main.go",
		"docs/examples/postgres/main.go",
		"docs/examples/sqlite/main.go",
		"examples/cli-app/main.go",
		"examples/http-service/main.go",
	}

	for _, example := range examples {
		if !strings.Contains(verifier, example) {
			t.Fatalf("workspace verifier does not compile-check %s", example)
		}
	}
}

func TestDocsExamplesUseRunnableFileCommands(t *testing.T) {
	t.Parallel()

	doc := readRepoFile(t, "docs", "examples", "README.md")
	staleCommands := []string{
		"go run ./docs/examples/logger",
		"go run ./docs/examples/config",
		"go run ./docs/examples/sqlite",
		"go run ./docs/examples/postgres",
		"go run ./docs/examples/cache",
		"go run ./docs/examples/openai",
	}

	for _, command := range staleCommands {
		if strings.Contains(doc, command) {
			t.Fatalf("docs/examples README contains stale package command %q; ignored examples must be run as files", command)
		}
	}
}

func parseGoWorkModules(content string) []string {
	var modules []string
	inUseBlock := false

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "use ("):
			inUseBlock = true
		case inUseBlock && trimmed == ")":
			inUseBlock = false
		case inUseBlock:
			modules = append(modules, strings.Trim(trimmed, `"`))
		case strings.HasPrefix(trimmed, "use "):
			modules = append(modules, strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "use ")), `"`))
		}
	}

	return modules
}

func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()

	path := filepath.Join(parts...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(data)
}
