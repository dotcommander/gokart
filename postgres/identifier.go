package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const postgresIdentifierMaxBytes = 63

// PostgresIdentifier carries a validated configured identifier and its quoted
// SQL form, which is safe to interpolate into PostgreSQL DDL and DML.
type PostgresIdentifier struct {
	Raw    string
	Quoted string
}

// NewPostgresIdentifier validates a single PostgreSQL identifier and returns
// its raw and double-quoted forms.
func NewPostgresIdentifier(name string) (PostgresIdentifier, error) {
	if err := validatePostgresIdentifierPart(name, true); err != nil {
		return PostgresIdentifier{}, err
	}
	return PostgresIdentifier{Raw: name, Quoted: `"` + name + `"`}, nil
}

// NewPostgresIndexIdentifier derives a deterministic index name bounded by
// PostgreSQL's 63-byte identifier limit.
func NewPostgresIndexIdentifier(tableName, suffix string) (PostgresIdentifier, error) {
	if err := validatePostgresIdentifierPart(tableName, true); err != nil {
		return PostgresIdentifier{}, fmt.Errorf("table name: %w", err)
	}
	if err := validatePostgresIdentifierPart(suffix, false); err != nil {
		return PostgresIdentifier{}, fmt.Errorf("suffix: %w", err)
	}

	raw := "idx_" + tableName + "_" + suffix
	if len(raw) > postgresIdentifierMaxBytes {
		sum := sha256.Sum256([]byte(raw))
		hash := hex.EncodeToString(sum[:])[:12]
		prefixLength := postgresIdentifierMaxBytes - len(hash) - 1
		raw = raw[:prefixLength] + "_" + hash
	}
	return NewPostgresIdentifier(raw)
}

func validatePostgresIdentifierPart(name string, enforceMax bool) error {
	if name == "" {
		return fmt.Errorf("identifier is empty")
	}
	if strings.TrimSpace(name) != name || strings.TrimSpace(name) == "" {
		return fmt.Errorf("identifier must not be blank or padded with whitespace")
	}
	if enforceMax && len(name) > postgresIdentifierMaxBytes {
		return fmt.Errorf("identifier %q exceeds PostgreSQL %d-byte limit", name, postgresIdentifierMaxBytes)
	}

	for i := range len(name) {
		char := name[i]
		if char >= 0x80 {
			return fmt.Errorf("identifier %q must be ASCII", name)
		}
		if i == 0 {
			if !isIdentifierStart(char) {
				return fmt.Errorf("identifier %q must start with a letter or underscore", name)
			}
			continue
		}
		if !isIdentifierContinue(char) {
			return fmt.Errorf("identifier %q contains invalid character %q", name, char)
		}
	}
	return nil
}

func isIdentifierStart(char byte) bool {
	return char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char == '_'
}

func isIdentifierContinue(char byte) bool {
	return isIdentifierStart(char) || char >= '0' && char <= '9'
}
