package main

import (
	"fmt"
	"os"

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

type errorOutput interface {
	setErrorFields(outcome commandOutcome, code commandErrorCode, exitCode int, errMsg string)
}

func (o *newCommandOutput) setErrorFields(outcome commandOutcome, code commandErrorCode, exitCode int, errMsg string) {
	o.Outcome = outcome
	o.ErrorCode = code
	o.ExitCode = exitCode
	o.Error = errMsg
}

func (o *addCommandOutput) setErrorFields(outcome commandOutcome, code commandErrorCode, exitCode int, errMsg string) {
	o.Outcome = outcome
	o.ErrorCode = code
	o.ExitCode = exitCode
	o.Error = errMsg
}

type commandFailureInfo struct {
	Code     commandErrorCode
	Outcome  commandOutcome
	ExitCode int
}

func emitCommandError(err error, jsonOutput bool, output errorOutput, fail commandFailureInfo) *commandError {
	cmdErr := &commandError{
		Err:      err,
		Code:     fail.Code,
		Outcome:  fail.Outcome,
		ExitCode: fail.ExitCode,
	}
	if jsonOutput && output != nil {
		output.setErrorFields(fail.Outcome, fail.Code, fail.ExitCode, err.Error())
		if emitErr := emitJSON(output); emitErr != nil {
			fmt.Fprintf(os.Stderr, "failed to write JSON output: %v\n", emitErr)
			return &commandError{
				Err:      fmt.Errorf("%w; failed to write JSON output: %v", err, emitErr), //nolint:errorlint // secondary error, primary already wrapped
				Code:     errorCodeJSONEncodeFailed,
				Outcome:  commandOutcomeFailure,
				ExitCode: exitCodeJSONEncodeFailed,
			}
		}
	}
	return cmdErr
}
