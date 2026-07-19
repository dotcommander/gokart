package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

const (
	appName               = "gokart"
	outcomeFailure        = "failure"
	outcomePartialSuccess = "partial_success"
	outcomeSuccess        = "success"
	jsonFlag              = "--json"
	configName            = "config"
)

type ProjectGenerator interface {
	Create(context.Context, generator.CreateRequest, generator.Runtime) (generator.CreateResult, error)
	Add(context.Context, generator.AddRequest, generator.Runtime) (generator.AddResult, error)
}

type Dependencies struct {
	Projects      ProjectGenerator
	Stdout        io.Writer
	Stderr        io.Writer
	Getwd         func() (string, error)
	UserConfigDir func() (string, error)
	BinaryPath    string
}

type executor struct {
	ctx     context.Context
	version string
	deps    Dependencies
}

type cli struct {
	VersionFlag kong.VersionFlag `name:"version" help:"Print version information and quit."`
	New         newCommand       `cmd:"" aliases:"create,init" help:"Create a new GoKart project."`
	Add         addCommand       `cmd:"" help:"Add integrations to a managed structured project."`
	Version     versionCommand   `cmd:"" help:"Print the version number."`
	Config      configCommand    `cmd:"" help:"Show gokart configuration."`
}

func Execute(ctx context.Context, args []string, version string, deps Dependencies) int {
	deps = normalizedDependencies(deps)
	args = legacyCompatibleArgs(args)
	if len(args) == 1 && args[0] == "--version" {
		_, _ = fmt.Fprintf(deps.Stdout, "%s version %s\n", appName, version)
		return 0
	}
	exec := &executor{ctx: ctx, version: version, deps: deps}
	root := cli{}
	root.New.exec, root.Add.exec = exec, exec
	root.Version.exec, root.Config.exec = exec, exec
	parser, err := newParser(&root, version, deps)
	var parsed *kong.Context
	parseFailed := false
	if err == nil {
		parsed, parseFailed, err = parseCommand(parser, args)
	}
	if err == nil && parsed != nil {
		root.New.changed = changedFlags(args)
		err = parsed.Run()
	}
	if err == nil {
		return 0
	}
	var classified *commandError
	if errors.As(err, &classified) {
		if !classified.emitted {
			writeDiagnostic(deps.Stderr, classified.Err)
		}
		return classified.exitCode
	}
	if hasJSONFlag(args) {
		var out any = createOutput{Outcome: outcomeFailure, ErrorCode: generator.ErrorInvalidArguments, ExitCode: 2, Error: err.Error()}
		if len(args) > 0 && args[0] == "add" {
			out = addOutput{Outcome: outcomeFailure, ErrorCode: generator.ErrorInvalidArguments, ExitCode: 2, Error: err.Error()}
		}
		if writeErr := emitJSON(deps.Stdout, out); writeErr != nil {
			writeDiagnostic(deps.Stderr, writeErr)
			return 8
		}
		return 2
	}
	writeDiagnostic(deps.Stderr, err)
	if parseFailed {
		return 2
	}
	return 1
}

func newParser(root *cli, version string, deps Dependencies, extra ...kong.Option) (*kong.Kong, error) {
	options := []kong.Option{
		kong.Name(appName),
		kong.Description(rootDescription),
		kong.Vars{"version": appName + " version " + version},
		kong.Writers(deps.Stdout, deps.Stderr),
		kong.UsageOnError(),
		kong.ConfigureHelp(helpOptions()),
	}
	return kong.New(root, append(options, extra...)...)
}

func helpOptions() kong.HelpOptions {
	return kong.HelpOptions{
		Compact:   true,
		Tree:      true,
		Summary:   true,
		FlagsLast: true,
	}
}

func normalizedDependencies(deps Dependencies) Dependencies {
	if deps.Stdout == nil {
		deps.Stdout = io.Discard
	}
	if deps.Stderr == nil {
		deps.Stderr = io.Discard
	}
	return deps
}

func parseCommand(parser *kong.Kong, args []string) (*kong.Context, bool, error) {
	parsed, err := parser.Parse(args)
	if err == nil {
		return parsed, false, nil
	}
	if len(args) > 1 || len(args) == 1 && args[0] != configName {
		return nil, true, err
	}
	var parseErr *kong.ParseError
	if !errors.As(err, &parseErr) {
		return nil, true, err
	}
	return nil, false, parseErr.Context.PrintUsage(false)
}

func legacyCompatibleArgs(args []string) []string {
	if len(args) == 1 && args[0] == "--version=false" {
		return nil
	}
	return args
}

func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == jsonFlag {
			return true
		}
		if value, ok := strings.CutPrefix(arg, jsonFlag+"="); ok {
			return value == "true"
		}
	}
	return false
}

type commandError struct {
	Err      error
	kind     generator.ErrorKind
	exitCode int
	emitted  bool
}

func (e *commandError) Error() string { return e.Err.Error() }
func (e *commandError) Unwrap() error { return e.Err }

func classify(err error) *commandError {
	var op *generator.OperationError
	if !errors.As(err, &op) {
		return &commandError{Err: err, exitCode: 1}
	}
	exits := map[generator.ErrorKind]int{
		generator.ErrorInvalidArguments: 2, generator.ErrorExistingFileConflict: 3,
		generator.ErrorVerifyFailed: 4, generator.ErrorTargetLocked: 5,
		generator.ErrorConfigInitFailed: 6, generator.ErrorScaffoldFailed: 7,
		generator.ErrorManifestNotFound: 9, generator.ErrorFlatModeUnsupported: 10,
	}
	code := exits[op.Kind]
	if code == 0 {
		code = 1
	}
	return &commandError{Err: err, kind: op.Kind, exitCode: code}
}

func emitJSON(w io.Writer, value any) error {
	return json.NewEncoder(w).Encode(value)
}

func writeDiagnostic(w io.Writer, err error) {
	_, _ = fmt.Fprintln(w, appName+":", err)
}

func changedFlags(args []string) map[string]bool {
	changed := make(map[string]bool)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			name, _, _ := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
			changed[name] = true
		}
	}
	return changed
}

const rootDescription = "GoKart scaffolds focused Go command-line applications."
