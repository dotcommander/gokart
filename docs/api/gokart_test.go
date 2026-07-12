package api_test

import (
	"os"
	"strings"
	"testing"
)

func TestRootAPIDocCoversCurrentExports(t *testing.T) {
	data, err := os.ReadFile("gokart.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, symbol := range []string{
		"ParseConfig", "MustParseConfig", "LoadConfig", "LoadConfigWithDefaults",
		"ConfigDir", "EnsureConfigDir", "SaveState", "LoadState", "StatePath",
	} {
		if !strings.Contains(doc, symbol) {
			t.Errorf("root API doc omits %s", symbol)
		}
	}
	for _, retired := range []string{"GetString", "GetInt", "GetFloat", "GetBool"} {
		if strings.Contains(doc, retired) {
			t.Errorf("root API doc contains retired %s", retired)
		}
	}
}
