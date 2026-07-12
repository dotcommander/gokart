package gokart_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dotcommander/gokart"
)

type testConfig struct {
	Host  string `mapstructure:"host"`
	Port  int    `mapstructure:"port"`
	Debug bool   `mapstructure:"debug"`
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestLoadConfigWithDefaults(t *testing.T) {
	t.Parallel()

	defaults := testConfig{
		Host:  "localhost",
		Port:  5432,
		Debug: false,
	}

	tests := []struct {
		name    string
		yaml    string // empty means no file
		wantErr bool
		want    testConfig
	}{
		{
			name: "partial config file retains defaults",
			yaml: "host: db.example.com\n",
			want: testConfig{
				Host:  "db.example.com",
				Port:  5432,  // retained from defaults
				Debug: false, // retained from defaults
			},
		},
		{
			name:    "no config file returns defaults",
			yaml:    "", // triggers nonexistent path
			wantErr: true,
			want:    defaults,
		},
		{
			name: "full config file overrides all defaults",
			yaml: "host: prod.example.com\nport: 3306\ndebug: true\n",
			want: testConfig{
				Host:  "prod.example.com",
				Port:  3306,
				Debug: true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var path string
			if tc.yaml != "" {
				path = writeTempYAML(t, tc.yaml)
			} else {
				// Use a path that cannot exist
				path = filepath.Join(t.TempDir(), "nonexistent.yaml")
			}

			got, err := gokart.LoadConfigWithDefaults(defaults, path)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoadConfigWithDefaults_MissingCandidateFallsThrough(t *testing.T) {
	t.Parallel()

	valid := writeTempYAML(t, "host: fallback.example.com\n")
	missing := filepath.Join(t.TempDir(), "missing.yaml")

	got, err := gokart.LoadConfigWithDefaults(testConfig{Port: 5432}, missing, valid)
	require.NoError(t, err)
	assert.Equal(t, testConfig{Host: "fallback.example.com", Port: 5432}, got)
}

func TestLoadConfigWithDefaults_InvalidCandidateDoesNotFallThrough(t *testing.T) {
	t.Parallel()

	invalid := writeTempYAML(t, "host: [unterminated\n")
	valid := writeTempYAML(t, "host: fallback.example.com\n")

	got, err := gokart.LoadConfigWithDefaults(testConfig{Host: "default"}, invalid, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), invalid)
	assert.Equal(t, testConfig{Host: "default"}, got)
}

func TestLoadConfigWithDefaults_UnreadableCandidateDoesNotFallThrough(t *testing.T) {
	t.Parallel()

	unreadable := t.TempDir()
	valid := writeTempYAML(t, "host: fallback.example.com\n")

	got, err := gokart.LoadConfigWithDefaults(testConfig{Host: "default"}, unreadable, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), unreadable)
	assert.Equal(t, testConfig{Host: "default"}, got)
}
