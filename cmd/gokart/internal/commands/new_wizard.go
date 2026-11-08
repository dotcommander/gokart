package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

const (
	layoutFlat       = "flat"
	layoutStructured = "structured"
)

type wizardAnswers struct {
	Project string
	Layout  string
	Example bool
	DB      string
	AI      bool
	Redis   bool
	Verify  bool
}

func wizardAnswersToRequest(a wizardAnswers, workingDir string, changed map[string]bool) generator.CreateRequest {
	req := generator.CreateRequest{
		Flat: a.Layout != layoutStructured, Structured: a.Layout == layoutStructured,
		DB: a.DB, AI: a.AI, Redis: a.Redis, Example: a.Example, Verify: a.Verify,
		ConfigScope: "auto", Changed: changed, WorkingDir: workingDir,
	}
	if req.DB == "" {
		req.DB = "none"
	}
	if name := strings.TrimSpace(a.Project); name != "" {
		req.Args = []string{name}
	}
	return req
}

func missingProjectError() error {
	return &generator.OperationError{
		Kind: generator.ErrorInvalidArguments,
		Err:  errors.New("missing project name: use `gokart new <project-name>` or run interactively"),
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok || f == nil {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func collectNewWizard(ctx context.Context, out io.Writer, workingDir string, changed map[string]bool) (generator.CreateRequest, error) {
	a := wizardAnswers{Layout: layoutFlat, Example: true, DB: "none"}
	run := func(fields ...huh.Field) error {
		return huh.NewForm(huh.NewGroup(fields...)).WithOutput(out).RunWithContext(ctx)
	}
	err := run(
		huh.NewInput().Title("Project name").Value(&a.Project).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("project name is required")
			}
			return nil
		}),
		huh.NewSelect[string]().Title("Layout").Options(
			huh.NewOption("Flat (single main.go)", layoutFlat),
			huh.NewOption("Structured (managed cmd/internal, supports integrations)", layoutStructured),
		).Value(&a.Layout),
		huh.NewConfirm().Title("Include example greet command?").Value(&a.Example),
	)
	if err != nil {
		return generator.CreateRequest{}, fmt.Errorf("new wizard: %w", err)
	}
	if a.Layout == layoutStructured {
		err = run(
			huh.NewSelect[string]().Title("Database backend").Options(
				huh.NewOption("None", "none"), huh.NewOption("SQLite", "sqlite"), huh.NewOption("PostgreSQL", "postgres"),
			).Value(&a.DB),
			huh.NewConfirm().Title("Include AI (OpenAI)?").Value(&a.AI),
			huh.NewConfirm().Title("Include Redis?").Value(&a.Redis),
		)
		if err != nil {
			return generator.CreateRequest{}, fmt.Errorf("new wizard: %w", err)
		}
	}
	a.Verify = a.Example
	if err := run(huh.NewConfirm().Title("Verify after generation (go test ./...)?").Value(&a.Verify)); err != nil {
		return generator.CreateRequest{}, fmt.Errorf("new wizard: %w", err)
	}
	req := wizardAnswersToRequest(a, workingDir, changed)
	if len(req.Args) == 0 {
		return generator.CreateRequest{}, missingProjectError()
	}
	return req, nil
}
