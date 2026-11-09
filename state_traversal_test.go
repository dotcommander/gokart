package gokart_test

import (
	"testing"

	"github.com/dotcommander/gokart"
)

func TestStateRejectsTraversalNames(t *testing.T) {
	t.Parallel()

	for _, app := range []string{"", ".", "..", "../evil", "/abs"} {
		if got := gokart.StatePath(app, "state.json"); got != "" {
			t.Errorf("StatePath(%q) = %q, want empty", app, got)
		}
		if err := gokart.SaveState(app, "state.json", map[string]string{"k": "v"}); err == nil {
			t.Errorf("SaveState app %q error = nil, want error", app)
		}
		if _, err := gokart.LoadState[map[string]string](app, "state.json"); err == nil {
			t.Errorf("LoadState app %q error = nil, want error", app)
		}
	}

	for _, file := range []string{"", ".", "..", "../evil.json", "/abs.json"} {
		if got := gokart.StatePath("myapp", file); got != "" {
			t.Errorf("StatePath file %q = %q, want empty", file, got)
		}
		if err := gokart.SaveState("myapp", file, map[string]string{"k": "v"}); err == nil {
			t.Errorf("SaveState file %q error = nil, want error", file)
		}
	}
}
