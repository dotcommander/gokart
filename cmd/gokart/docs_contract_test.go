package main

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestGeneratorDocsMatchLiveCobraFlagTables(t *testing.T) {
	t.Parallel()
	docBytes, err := os.ReadFile("../../docs/components/generator.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(docBytes)
	assertDocumentedFlags(t, markdownSection(t, doc, "### `gokart new` — Full Flag Table", "### `gokart add`"), newNewCommand(), map[string]string{
		newFlagModule: "project name",
	})
	assertDocumentedFlags(t, markdownSection(t, doc, "### `gokart add` — Full Flag Table", "## See Also"), newAddCommand(), nil)
}

func assertDocumentedFlags(t *testing.T, table string, cmd *cobra.Command, semanticDefaults map[string]string) {
	t.Helper()
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		rowPrefix := "| `--" + flag.Name + "` |"
		var row string
		for _, line := range strings.Split(table, "\n") {
			if strings.HasPrefix(line, rowPrefix) {
				row = line
				break
			}
		}
		if row == "" {
			t.Errorf("%s docs omit live flag --%s", cmd.Use, flag.Name)
			return
		}
		wantDefault := semanticDefaults[flag.Name]
		if wantDefault == "" {
			wantDefault = strings.TrimSuffix(flag.DefValue, "0s")
		}
		if !strings.Contains(row, "| "+wantDefault+" |") && !strings.Contains(row, "| `"+wantDefault+"` |") {
			t.Errorf("%s row for --%s omits live default %q: %s", cmd.Use, flag.Name, wantDefault, row)
		}
	})
}

func markdownSection(t *testing.T, doc, start, end string) string {
	t.Helper()
	_, section, ok := strings.Cut(doc, start)
	if !ok {
		t.Fatalf("missing docs section %q", start)
	}
	section, _, ok = strings.Cut(section, end)
	if !ok {
		t.Fatalf("missing docs section terminator %q", end)
	}
	return section
}
