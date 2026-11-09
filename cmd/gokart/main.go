package main

import (
	"context"
	"os"
)

// devGokartVersion marks a version built from source (no -X ldflags override).
const devGokartVersion = "dev"

var gokartVersion = devGokartVersion

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Args[0], os.Stdout, os.Stderr))
}
