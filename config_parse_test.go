package gokart_test

import (
	"testing"
	"time"

	"github.com/dotcommander/gokart"
)

func TestParseConfigDefaultsRequiredAndEmbeddedFields(t *testing.T) {
	type Common struct {
		Timeout time.Duration `config:"timeout" default:"3000000000"`
	}
	type config struct {
		Common
		APIKey string `config:"api_key" required:"true"`
		Port   int32  `config:"port" default:"5432"`
		Debug  bool   `config:"debug" default:"true"`
	}

	got, err := gokart.ParseConfig[config](map[string]any{"api_key": "secret", "port": int(6432)})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if got.APIKey != "secret" || got.Port != 6432 || !got.Debug || got.Timeout != 3*time.Second {
		t.Fatalf("ParseConfig() = %+v", got)
	}
}

func TestParseConfigRejectsInvalidShapesAndConversions(t *testing.T) {
	type config struct {
		Port int `config:"port" required:"true"`
	}

	if _, err := gokart.ParseConfig[string](nil); err == nil {
		t.Fatal("ParseConfig[string]() error = nil")
	}
	if _, err := gokart.ParseConfig[config](nil); err == nil {
		t.Fatal("missing required field error = nil")
	}
	if _, err := gokart.ParseConfig[config](map[string]any{"port": "5432"}); err == nil {
		t.Fatal("string-to-int conversion error = nil")
	}
}

func TestMustParseConfigPanicsOnInvalidConfig(t *testing.T) {
	type config struct {
		Name string `config:"name" required:"true"`
	}
	defer func() {
		if recover() == nil {
			t.Fatal("MustParseConfig() did not panic")
		}
	}()
	gokart.MustParseConfig[config](nil)
}
