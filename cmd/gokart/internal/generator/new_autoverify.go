package generator

import (
	"strconv"
	"strings"
)

// resolveAutoVerify decides whether the normal scaffold path runs verification.
// Verification is default-on; --no-verify and GOKART_AUTO_VERIFY=0 opt out.
func resolveAutoVerify(_ newRequest, explicitVerify, noVerify bool, lookupEnv func(string) (string, bool), _ *[]string) bool {
	if explicitVerify {
		return true
	}
	if noVerify {
		return false
	}
	if autoVerifyEnvDisabled(lookupEnv) {
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
