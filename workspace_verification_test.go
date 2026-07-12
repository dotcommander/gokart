package gokart_test

import (
	"os"
	"path/filepath"
	"slices"
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

func TestWorkspaceReleaseAndDocsModuleInventoriesAgree(t *testing.T) {
	t.Parallel()
	workspace := parseGoWorkModules(readRepoFile(t, "go.work"))
	for i := range workspace {
		workspace[i] = strings.TrimPrefix(workspace[i], "./")
	}
	workspace = slices.DeleteFunc(workspace, func(module string) bool { return module == "." })
	slices.Sort(workspace)

	just := readRepoFile(t, "justfile")
	line := strings.SplitN(strings.SplitN(just, "modules := \"", 2)[1], "\"", 2)[0]
	release := strings.Fields(line)
	slices.Sort(release)
	if !slices.Equal(workspace, release) {
		t.Fatalf("workspace modules %v do not match release modules %v", workspace, release)
	}

	readme := readRepoFile(t, "README.md")
	for _, module := range release {
		if !strings.Contains(readme, "`gokart/"+strings.TrimPrefix(module, "cmd/")+"`") && module != "cmd/gokart" {
			t.Fatalf("README module inventory omits %s", module)
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

func TestStandaloneModulesAreVerifiedWithoutWorkspaceReplacements(t *testing.T) {
	t.Parallel()

	verifier := readRepoFile(t, "scripts", "verify-workspace.sh")
	required := []string{
		"standalone)",
		"verify_standalone_modules",
		"GOWORK=off go test -modfile=\"$modfile\" -mod=readonly ./...",
		"compile_ignored_examples\n        verify_standalone_modules",
	}
	for _, fragment := range required {
		if !strings.Contains(verifier, fragment) {
			t.Fatalf("workspace verifier is missing standalone-module gate %q", fragment)
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
