package gokart

import "testing"

func TestConfigMapHelpers(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"string":  "value",
		"int":     42,
		"int64":   int64(43),
		"float64": float64(44.75),
		"float32": float32(0.75),
		"bool":    true,
	}

	if got := GetString(config, "string", "default"); got != "value" {
		t.Fatalf("GetString() = %q, want value", got)
	}
	if got := GetString(config, "int", "default"); got != "default" {
		t.Fatalf("GetString(wrong type) = %q, want default", got)
	}
	for key, want := range map[string]int{"int": 42, "int64": 43, "float64": 44} {
		if got := GetInt(config, key, -1); got != want {
			t.Fatalf("GetInt(%q) = %d, want %d", key, got, want)
		}
	}
	for key, want := range map[string]float32{"float32": 0.75, "float64": 44.75, "int": 42} {
		if got := GetFloat(config, key, -1); got != want {
			t.Fatalf("GetFloat(%q) = %v, want %v", key, got, want)
		}
	}
	if got := GetBool(config, "bool", false); !got {
		t.Fatal("GetBool() = false, want true")
	}
}

func TestConfigMapHelpersUseDefaults(t *testing.T) {
	t.Parallel()

	var config map[string]any
	if got := GetString(config, "missing", "fallback"); got != "fallback" {
		t.Fatalf("GetString() = %q, want fallback", got)
	}
	if got := GetInt(config, "missing", 7); got != 7 {
		t.Fatalf("GetInt() = %d, want 7", got)
	}
	if got := GetFloat(config, "missing", 0.5); got != 0.5 {
		t.Fatalf("GetFloat() = %v, want 0.5", got)
	}
	if got := GetBool(config, "missing", true); !got {
		t.Fatal("GetBool() = false, want true")
	}
}
