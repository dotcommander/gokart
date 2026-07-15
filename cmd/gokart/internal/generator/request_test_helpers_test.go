package generator

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type NewCommand = CreateRequest

func newNewCommandForTest() *CreateRequest {
	return &CreateRequest{DB: "none", ConfigScope: configScopeAuto, VerifyTimeout: defaultVerifyTimeout, Changed: map[string]bool{}, lookupEnv: os.LookupEnv}
}

func setCreateFlag(command *CreateRequest, name, value string) error { //nolint:gocyclo // test adapter intentionally mirrors the complete CLI flag set
	command.Changed[name] = true
	b, err := strconv.ParseBool(value)
	switch name {
	case newFlagFlat:
		command.Flat = b
	case newFlagStructured:
		command.Structured = b
	case newFlagAI:
		command.AI = b
	case newFlagRedis:
		command.Redis = b
	case newFlagExample:
		command.Example = b
	case newFlagLocal:
		command.Local = b
	case newFlagGlobal:
		command.Global = b
	case newFlagDryRun:
		command.DryRun = b
	case newFlagForce:
		command.Force = b
	case newFlagSkipExisting:
		command.SkipExisting = b
	case newFlagNoManifest:
		command.NoManifest = b
	case newFlagVerify:
		command.Verify = b
	case newFlagNoVerify:
		command.NoVerify = b
	case newFlagVerifyOnly:
		command.VerifyOnly = b
	case newFlagDB:
		command.DB = value
		return nil
	case newFlagModule:
		command.Module = value
		return nil
	case newFlagConfigScope:
		command.ConfigScope = value
		return nil
	case newFlagVerifyTimeout:
		d, parseErr := time.ParseDuration(value)
		command.VerifyTimeout = d
		return parseErr
	default:
		return fmt.Errorf("unknown flag %q", name)
	}
	return err
}

func mustSetFlagTrue(t interface {
	Helper()
	Fatalf(string, ...any)
}, cmd *CreateRequest, name string) {
	t.Helper()
	if err := setCreateFlag(cmd, name, "true"); err != nil {
		t.Fatalf("set %s: %v", name, err)
	}
}
