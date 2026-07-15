package main

import (
	"context"
	"os"
)

var gokartVersion = "dev"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Args[0], os.Stdout, os.Stderr))
}
