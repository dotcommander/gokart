package main

import (
	"fmt"
	"strings"
)

// resolveDB converts the --db string flag to UseSQLite/UsePostgres booleans.
// Valid values: "sqlite", "postgres", "none" (default). Returns an error for
// unrecognised values, enforcing the single-choice contract at parse time.
func resolveDB(db string) (useSQLite, usePostgres bool, err error) {
	switch strings.ToLower(strings.TrimSpace(db)) {
	case "sqlite":
		return true, false, nil
	case "postgres":
		return false, true, nil
	case "none", "":
		return false, false, nil
	default:
		return false, false, fmt.Errorf("invalid --db %q (valid values: sqlite, postgres, none)", db)
	}
}
