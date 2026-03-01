package gokart_test

import (
	"bytes"
	"testing"

	"github.com/dotcommander/gokart"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config gokart.LogConfig
	}{
		{
			name:   "defaults",
			config: gokart.LogConfig{},
		},
		{
			name:   "debug json",
			config: gokart.LogConfig{Level: "debug", Format: "json"},
		},
		{
			name:   "warn text",
			config: gokart.LogConfig{Level: "warn", Format: "text"},
		},
		{
			name:   "custom output",
			config: gokart.LogConfig{Output: &bytes.Buffer{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := gokart.NewLogger(tt.config)
			if log == nil {
				t.Fatal("expected logger, got nil")
			}
			log.Info("test message", "key", "value")
		})
	}
}

func ExampleNewLogger() {
	log := gokart.NewLogger(gokart.LogConfig{
		Level:  "info",
		Format: "json",
	})
	log.Info("server started", "port", 8080)
}
