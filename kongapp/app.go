// Package kongapp provides a small execution lifecycle for Kong command trees.
// Applications retain ownership of their command structs, dependencies, config,
// signal handling, and process exit policy.
package kongapp

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/alecthomas/kong"
)

// Streams contains caller-owned process streams. Commands can request Streams
// as a Run method dependency; Kong's own output is routed to Out and Err.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// Cleanup releases resources acquired by an Initializer.
type Cleanup func() error

// Initializer runs after successful parsing and before the selected command.
// The parsed command, populated command tree, and effective arguments are
// available for application-specific setup. Returning a Cleanup makes teardown
// part of the same lifecycle even when command execution fails.
type Initializer[T any] func(
	ctx context.Context,
	parsed *kong.Context,
	tree *T,
	args []string,
) (Cleanup, error)

// Config defines infrastructure policy for one Kong command tree. Application
// and domain policy stay in the tree, injected options, and Initializer.
type Config[T any] struct {
	Name        string
	Description string
	Vars        kong.Vars
	Help        kong.HelpOptions
	DefaultArgs []string
	Options     []kong.Option
	Initialize  Initializer[T]
}

// Execute parses and runs a consumer-owned Kong command tree. args excludes the
// process name. The caller owns cancellation, streams, dependency bindings, and
// conversion of returned errors into process exit statuses.
func Execute[T any](
	ctx context.Context,
	args []string,
	streams Streams,
	tree *T,
	cfg Config[T],
) (err error) {
	if ctx == nil {
		return errors.New("kongapp: context is required")
	}
	if tree == nil {
		return errors.New("kongapp: command tree is required")
	}
	if cfg.Name == "" {
		return errors.New("kongapp: application name is required")
	}
	if streams.Out == nil {
		return errors.New("kongapp: stdout is required")
	}
	if streams.Err == nil {
		return errors.New("kongapp: stderr is required")
	}

	effectiveArgs := append([]string(nil), args...)
	if len(effectiveArgs) == 0 && len(cfg.DefaultArgs) > 0 {
		effectiveArgs = append([]string(nil), cfg.DefaultArgs...)
	}

	options := []kong.Option{
		kong.Name(cfg.Name),
		kong.Writers(streams.Out, streams.Err),
		kong.BindTo(ctx, (*context.Context)(nil)),
		kong.Bind(streams),
		kong.ConfigureHelp(cfg.Help),
	}
	if cfg.Description != "" {
		options = append(options, kong.Description(cfg.Description))
	}
	if cfg.Vars != nil {
		options = append(options, cfg.Vars)
	}
	options = append(options, cfg.Options...)

	parser, err := kong.New(tree, options...)
	if err != nil {
		return fmt.Errorf("kongapp: build parser: %w", err)
	}
	exited := false
	parser.Exit = func(int) { exited = true }
	parsed, err := parser.Parse(effectiveArgs)
	if exited {
		return nil
	}
	if err != nil {
		return err
	}

	if cfg.Initialize != nil {
		cleanup, initErr := cfg.Initialize(ctx, parsed, tree, effectiveArgs)
		if cleanup != nil {
			defer func() {
				err = errors.Join(err, cleanup())
			}()
		}
		if initErr != nil {
			return initErr
		}
	}

	return parsed.Run()
}
