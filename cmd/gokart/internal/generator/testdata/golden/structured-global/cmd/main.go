package main

import (
	"context"
	"fmt"
	"os"

	"github.com/example/demo/internal/commands"
)

// version is set via ldflags: -X main.version=v1.0.0
var version = "dev"

func main() {
	if err := commands.Execute(context.Background(), version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
