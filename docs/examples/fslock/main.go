//go:build ignore

// Example: Cross-process coordination with gokart/fslock.
//
// Advisory locks only exclude processes that also acquire the same lock.
// Unix only; other platforms return fslock.ErrUnsupported.
//
// Run with: go run docs/examples/fslock/main.go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/dotcommander/gokart/fslock"
)

func main() {
	lockPath := fmt.Sprintf("%s/example.lock", os.TempDir())

	// nonblocking=true: fail fast with ErrLocked instead of waiting.
	lock, err := fslock.Acquire(lockPath, true)
	if err != nil {
		if errors.Is(err, fslock.ErrLocked) {
			fmt.Println("another process holds the lock; exiting")
			return
		}
		log.Fatal(err)
	}
	defer func() { _ = lock.Unlock() }() // idempotent; Close is an alias

	// Critical section: only one process is here at a time.
	fmt.Println("holding the lock; doing exclusive work…")

	// To wait instead of failing, pass false:
	//   lock, err := fslock.Acquire(lockPath, false)
	_ = lock
}
