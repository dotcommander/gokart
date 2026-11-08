//go:build !unix

package fslock

import "os"

func lockFile(*os.File, bool) error { return ErrUnsupported }

func unlockFile(*os.File) error { return nil }
