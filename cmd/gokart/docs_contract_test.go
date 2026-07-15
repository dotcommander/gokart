package main

import (
	"os"
	"strings"
	"testing"
)

func TestGeneratorDocsKeepConfigOutputExample(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../docs/components/generator.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	for _, line := range []string{"Version:     v0.13.0", "Config dir:  /Users/you/Library/Application Support", "Binary:      /Users/you/go/bin/gokart"} {
		if !strings.Contains(doc, line) {
			t.Errorf("docs omit config output %q", line)
		}
	}
}
