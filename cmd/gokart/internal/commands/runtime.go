package commands

import "github.com/dotcommander/gokart/cmd/gokart/internal/generator"

func (e *executor) runtime() generator.Runtime {
	return generator.Runtime{Stdout: e.deps.Stdout, Stderr: e.deps.Stderr, Verbose: false}
}
