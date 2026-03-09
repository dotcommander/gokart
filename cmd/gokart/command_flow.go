package main

import (
	"fmt"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"
)

func configureJSONCommand(cmd *cobra.Command, enabled bool) {
	if !enabled || cmd == nil {
		return
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
}

func printWarnings(jsonOutput bool, warnings []string) {
	if jsonOutput {
		return
	}

	for _, warning := range warnings {
		cli.Warning("%s", warning)
	}
}

func emitCommandJSON(v any) error {
	if err := emitJSON(v); err != nil {
		return &commandError{
			Err:      fmt.Errorf("encode JSON output: %w", err),
			Code:     errorCodeJSONEncodeFailed,
			Outcome:  commandOutcomeFailure,
			ExitCode: exitCodeJSONEncodeFailed,
		}
	}

	return nil
}
