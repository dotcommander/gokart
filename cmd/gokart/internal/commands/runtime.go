package commands

import "github.com/dotcommander/gokart/cmd/gokart/internal/generator"

func (e *executor) runtime(jsonOutput bool) generator.Runtime {
	runtime := generator.Runtime{Stdout: e.deps.Stdout, Stderr: e.deps.Stderr, Verbose: !jsonOutput}
	if !jsonOutput {
		runtime.Report = func(event generator.Event) {
			writeOutputln(e.deps.Stdout, event.Message)
		}
	}
	return runtime
}
