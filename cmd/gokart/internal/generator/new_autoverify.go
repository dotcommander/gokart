package generator

import (
	"strconv"
	"strings"
)

// resolveAutoVerify decides whether the normal scaffold path runs verification.
// Verification is default-on; --no-verify and GOKART_AUTO_VERIFY=0 opt out.
// Network-dependent integrations (--db postgres, --redis) skip auto-verify unless
// the user explicitly passed --verify, since `go test ./...` on those scaffolds
// can hang against an absent DB/Redis. Explicit --verify always wins.
func resolveAutoVerify(req newRequest, explicitVerify, noVerify bool, lookupEnv func(string) (string, bool), warnings *[]string) bool {
	if explicitVerify {
		return true
	}
	if noVerify {
		return false
	}
	if autoVerifyEnvDisabled(lookupEnv) {
		return false
	}
	if req.UsePostgres || req.UseRedis {
		if warnings != nil {
			*warnings = append(*warnings, "skipping automatic verification for network-dependent integration (--db postgres/--redis); rerun with --verify to force it")
		}
		return false
	}
	return true
}

// autoVerifyEnvDisabled reports whether GOKART_AUTO_VERIFY is set to a falsey
// value (0/false), used by CI flows that build separately.
func autoVerifyEnvDisabled(lookupEnv func(string) (string, bool)) bool {
	if lookupEnv == nil {
		return false
	}
	raw, ok := lookupEnv("GOKART_AUTO_VERIFY")
	if !ok {
		return false
	}
	if v, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
		return !v
	}
	return false
}
