package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCreateModeMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mutate  func(*CreateRequest)
		want    string
		wantErr bool
	}{
		{"plain", func(*CreateRequest) {}, modeFlat, false},
		{"example", func(r *CreateRequest) { r.Example = true }, modeFlat, false},
		{"global", func(r *CreateRequest) { r.Global = true }, modeFlat, false},
		{"structured", func(r *CreateRequest) { r.Structured = true }, modeStructured, false},
		{"sqlite", func(r *CreateRequest) { r.DB = "sqlite" }, modeStructured, false},
		{"flat structured", func(r *CreateRequest) { r.Flat, r.Structured = true, true }, "", true},
		{"flat integration", func(r *CreateRequest) { r.Flat, r.AI = true, true }, "", true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := newNewCommandForTest()
			req.Args = []string{filepath.Join(t.TempDir(), "demo")}
			req.NoVerify = true
			tc.mutate(req)
			got, err := buildNewRequest(req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			if err == nil && got.Mode != tc.want {
				t.Fatalf("mode=%q want=%q", got.Mode, tc.want)
			}
		})
	}
}

func TestVerifyOnlyIgnoresConflictingLayoutFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	req := newNewCommandForTest()
	req.Args = []string{dir}
	req.VerifyOnly, req.Flat, req.Structured, req.AI = true, true, true, true
	req.Changed = map[string]bool{newFlagFlat: true, newFlagStructured: true, newFlagAI: true}
	got, err := buildNewRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 1 {
		t.Fatalf("warnings=%v", got.Warnings)
	}
}

func TestLegacyGreetAppFieldDetectionIsExact(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "greet.go")
	for _, tc := range []struct {
		source string
		want   bool
	}{{"package commands\nimport \"x/app\"\ntype GreetCommand struct{ App *app.Context }\n", true}, {"package commands\ntype GreetCommand struct{ App string }\n", false}} {
		if err := os.WriteFile(path, []byte(tc.source), 0644); err != nil {
			t.Fatal(err)
		}
		if got := hasLegacyGreetAppField(path); got != tc.want {
			t.Fatalf("got=%v want=%v source=%q", got, tc.want, tc.source)
		}
	}
}
