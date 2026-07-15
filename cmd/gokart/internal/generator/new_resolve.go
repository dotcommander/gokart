package generator

import (
	"fmt"
	"strings"
)

// resolveDB converts the --db string flag to UseSQLite/UsePostgres booleans.
// Valid values: "sqlite", "postgres", "none" (default). Returns an error for
// unrecognised values, enforcing the single-choice contract at parse time.
func resolveDB(db string) (useSQLite, usePostgres bool, err error) {
	switch strings.ToLower(strings.TrimSpace(db)) {
	case integrationSQLite:
		return true, false, nil
	case integrationPostgres:
		return false, true, nil
	case integrationNone, "":
		return false, false, nil
	default:
		return false, false, fmt.Errorf("invalid --db %q (valid values: sqlite, postgres, none)", db)
	}
}
