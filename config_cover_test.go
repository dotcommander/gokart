package gokart_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dotcommander/gokart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigFirstFoundWins(t *testing.T) {
	t.Parallel()

	type cfg struct {
		Host string `mapstructure:"host"`
	}
	a := filepath.Join(t.TempDir(), "a.yaml")
	b := filepath.Join(t.TempDir(), "b.yaml")
	require.NoError(t, os.WriteFile(a, []byte("host: first\n"), 0600))
	require.NoError(t, os.WriteFile(b, []byte("host: second\n"), 0600))

	got, err := gokart.LoadConfig[cfg](a, b)
	require.NoError(t, err)
	assert.Equal(t, "first", got.Host)
}

func TestLoadConfigNoPathsUsesZeroDefaults(t *testing.T) {
	t.Parallel()

	type cfg struct {
		Port int `mapstructure:"port"`
	}
	got, err := gokart.LoadConfig[cfg]()
	require.NoError(t, err)
	assert.Equal(t, 0, got.Port)
}

func TestParseConfigNumericAndUnsupportedDefaults(t *testing.T) {
	t.Parallel()

	type numeric struct {
		Count int64   `config:"count" default:"42"`
		Ratio float64 `config:"ratio" default:"1.5"`
	}
	got, err := gokart.ParseConfig[numeric](nil)
	require.NoError(t, err)
	assert.Equal(t, int64(42), got.Count)
	assert.InEpsilon(t, 1.5, got.Ratio, 0.001)

	type badDefault struct {
		Port int `config:"port" default:"not-a-number"`
	}
	if _, err := gokart.ParseConfig[badDefault](nil); err == nil {
		t.Error("invalid int default must error")
	}

	// Uint fields have no string-default support by design.
	type uintField struct {
		Count uint64 `config:"count" default:"42"`
	}
	if _, err := gokart.ParseConfig[uintField](nil); err == nil {
		t.Error("uint default must error as unsupported")
	}

	type unsupported struct {
		Items []string `config:"items" default:"x"`
	}
	if _, err := gokart.ParseConfig[unsupported](nil); err == nil {
		t.Error("unsupported slice default must error")
	}
}
