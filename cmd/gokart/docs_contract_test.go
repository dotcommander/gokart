package main

import (
	"os"
	"strings"
	"testing"
)

func TestGeneratorDocsMatchKongFlagTables(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../docs/components/generator.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	for _, flag := range []string{"flat", "structured", "module", "db", "ai", "redis", "example", "local", "global", "config-scope", "dry-run", "force", "skip-existing", "no-manifest", "verify", "no-verify", "verify-only", "verify-timeout", "json"} {
		if !strings.Contains(doc, "`--"+flag+"`") {
			t.Errorf("docs omit --%s", flag)
		}
	}
	for _, flag := range []string{"dry-run", "force", "json", "verify", "verify-timeout"} {
		if strings.Count(doc, "`--"+flag+"`") < 2 {
			t.Errorf("docs omit add --%s", flag)
		}
	}
	for _, line := range []string{"Version:     v0.12.0", "Config dir:  /Users/you/Library/Application Support", "Binary:      /Users/you/go/bin/gokart"} {
		if !strings.Contains(doc, line) {
			t.Errorf("docs omit config output %q", line)
		}
	}
}
