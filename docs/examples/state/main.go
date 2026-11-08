//go:build ignore

// Example: Small CLI state with gokart's SaveState/LoadState.
//
// State is the platform user config directory (os.UserConfigDir), not the
// project directory. Use a database for concurrent or queryable data.
// Never store secrets here: files are human-readable (0600) but unencrypted.
//
// Run with: go run docs/examples/state/main.go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/dotcommander/gokart"
)

type AppState struct {
	LastFile   string   `json:"last_file"`
	Recent     []string `json:"recent_files"`
	VisitCount int      `json:"visit_count"`
}

func main() {
	state, err := gokart.LoadState[AppState]("example", "state.json")
	if errors.Is(err, os.ErrNotExist) {
		// First run: defaults.
		state = AppState{Recent: []string{}}
	} else if err != nil {
		log.Fatal(err)
	}

	state.LastFile = "/new/file.txt"
	state.VisitCount++
	state.Recent = append(state.Recent, "/new/file.txt")
	if len(state.Recent) > 10 {
		state.Recent = state.Recent[1:]
	}

	// Atomic publish: indented JSON, 0600, via temp file + rename.
	if err := gokart.SaveState("example", "state.json", state); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("visited %d times; state at %s\n", state.VisitCount, gokart.StatePath("example", "state.json"))
}
