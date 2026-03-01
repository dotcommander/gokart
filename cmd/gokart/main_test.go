package main

import (
	"path/filepath"
	"testing"
)

func TestParseNewInvocation(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantPreset string
		wantArg    string
		wantErr    bool
	}{
		{name: "legacy form", args: []string{"myapp"}, wantPreset: defaultPreset, wantArg: "myapp"},
		{name: "preset form", args: []string{"cli", "myapp"}, wantPreset: defaultPreset, wantArg: "myapp"},
		{name: "ambiguous preset only", args: []string{"cli"}, wantErr: true},
		{name: "unknown preset", args: []string{"api", "myapp"}, wantErr: true},
		{name: "missing args", args: []string{}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preset, arg, err := parseNewInvocation(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseNewInvocation() error = %v", err)
			}

			if preset != tc.wantPreset || arg != tc.wantArg {
				t.Fatalf("got preset=%q arg=%q, want preset=%q arg=%q", preset, arg, tc.wantPreset, tc.wantArg)
			}
		})
	}
}

func TestResolveUseGlobal(t *testing.T) {
	tests := []struct {
		name        string
		flat        bool
		local       bool
		global      bool
		scope       string
		want        bool
		wantWarning bool
		wantErr     bool
	}{
		{name: "structured defaults global", scope: configScopeAuto, want: true},
		{name: "structured local legacy", local: true, scope: configScopeAuto, want: false},
		{name: "flat defaults local", flat: true, scope: configScopeAuto, want: false},
		{name: "flat legacy global", flat: true, global: true, scope: configScopeAuto, want: true},
		{name: "explicit local", scope: configScopeLocal, want: false},
		{name: "explicit global", scope: configScopeGlobal, want: true},
		{name: "conflicting legacy flags", local: true, global: true, scope: configScopeAuto, wantErr: true},
		{name: "scope with legacy flags", local: true, scope: configScopeGlobal, wantErr: true},
		{name: "flat local warning", flat: true, local: true, scope: configScopeAuto, wantWarning: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, warnings, err := resolveUseGlobal(tc.flat, tc.local, tc.global, tc.scope)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveUseGlobal() error = %v", err)
			}

			if got != tc.want {
				t.Fatalf("resolveUseGlobal() = %v, want %v", got, tc.want)
			}

			hasWarning := len(warnings) > 0
			if hasWarning != tc.wantWarning {
				t.Fatalf("resolveUseGlobal() warnings=%v, expected warning=%v", warnings, tc.wantWarning)
			}
		})
	}
}

func TestResolveExistingFilePolicy(t *testing.T) {
	policy, err := resolveExistingFilePolicy(false, false)
	if err != nil {
		t.Fatalf("resolveExistingFilePolicy() error = %v", err)
	}
	if policy != ExistingFilePolicyFail {
		t.Fatalf("policy = %q, want %q", policy, ExistingFilePolicyFail)
	}

	policy, err = resolveExistingFilePolicy(true, false)
	if err != nil {
		t.Fatalf("resolveExistingFilePolicy(force) error = %v", err)
	}
	if policy != ExistingFilePolicyOverwrite {
		t.Fatalf("policy = %q, want %q", policy, ExistingFilePolicyOverwrite)
	}

	policy, err = resolveExistingFilePolicy(false, true)
	if err != nil {
		t.Fatalf("resolveExistingFilePolicy(skip) error = %v", err)
	}
	if policy != ExistingFilePolicySkip {
		t.Fatalf("policy = %q, want %q", policy, ExistingFilePolicySkip)
	}

	if _, err := resolveExistingFilePolicy(true, true); err == nil {
		t.Fatal("expected conflict error for --force + --skip-existing")
	}
}

func TestNormalizeProjectArg(t *testing.T) {
	name, dir, err := normalizeProjectArg("myapp")
	if err != nil {
		t.Fatalf("normalizeProjectArg() error = %v", err)
	}
	if name != "myapp" {
		t.Fatalf("name = %q, want myapp", name)
	}
	if dir != filepath.Join(".", "myapp") {
		t.Fatalf("dir = %q, want %q", dir, filepath.Join(".", "myapp"))
	}

	if _, _, err := normalizeProjectArg("."); err == nil {
		t.Fatal("expected error for invalid project name")
	}

	if _, _, err := normalizeProjectArg("bad name"); err == nil {
		t.Fatal("expected error for space in project name")
	}
}

func TestValidateModulePath(t *testing.T) {
	if err := validateModulePath("github.com/acme/myapp"); err != nil {
		t.Fatalf("validateModulePath(valid) error = %v", err)
	}

	if err := validateModulePath("myapp"); err != nil {
		t.Fatalf("validateModulePath(simple) error = %v", err)
	}

	if err := validateModulePath("github.com/acme/my app"); err == nil {
		t.Fatal("expected error for whitespace in module path")
	}

	if err := validateModulePath("github.com//acme"); err == nil {
		t.Fatal("expected error for empty module path segment")
	}
}
