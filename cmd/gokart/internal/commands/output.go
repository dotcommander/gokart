package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

type nextStep struct {
	Dir     string   `json:"dir,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type createOutput struct {
	Outcome            string                       `json:"outcome,omitempty"`
	ErrorCode          generator.ErrorKind          `json:"error_code,omitempty"`
	ExitCode           int                          `json:"exit_code"`
	Preset             string                       `json:"preset,omitempty"`
	Mode               string                       `json:"mode,omitempty"`
	ProjectName        string                       `json:"project_name,omitempty"`
	TargetDir          string                       `json:"target_dir,omitempty"`
	Module             string                       `json:"module,omitempty"`
	ConfigScope        string                       `json:"config_scope,omitempty"`
	UseGlobal          bool                         `json:"use_global"`
	DryRun             bool                         `json:"dry_run"`
	WriteManifest      bool                         `json:"write_manifest"`
	VerifyRequested    bool                         `json:"verify_requested"`
	VerifyOnly         bool                         `json:"verify_only"`
	VerifyRan          bool                         `json:"verify_ran"`
	VerifyPassed       bool                         `json:"verify_passed"`
	Checks             []generator.CheckResult      `json:"checks,omitempty"`
	ExistingFilePolicy generator.ExistingFilePolicy `json:"existing_file_policy,omitempty"`
	Warnings           []string                     `json:"warnings,omitempty"`
	Conflicts          []string                     `json:"conflicts,omitempty"`
	Result             *generator.ApplyResult       `json:"result,omitempty"`
	Next               *nextStep                    `json:"next,omitempty"`
	NextCommand        string                       `json:"next_command,omitempty"`
	NextSteps          []string                     `json:"next_steps,omitempty"`
	Error              string                       `json:"error,omitempty"`
}

func createOutputFrom(r generator.CreateResult) createOutput {
	out := createOutput{Preset: r.Preset, Mode: r.Mode, ProjectName: r.ProjectName, TargetDir: r.TargetDir,
		Module: r.Module, ConfigScope: r.ConfigScope, UseGlobal: r.UseGlobal, DryRun: r.DryRun,
		WriteManifest: r.WriteManifest, VerifyRequested: r.VerifyRequested, VerifyOnly: r.VerifyOnly,
		VerifyRan: r.VerifyRan, VerifyPassed: r.VerifyPassed, ExistingFilePolicy: r.ExistingFilePolicy,
		Checks: r.Checks, Warnings: r.Warnings, Conflicts: r.Conflicts, Result: r.Result,
		NextSteps: append([]string(nil), r.NextSteps...)}
	if r.NextCommand != "" {
		out.Next = &nextStep{Dir: r.NextDir, Command: r.NextCommand, Args: r.NextArgs}
		out.NextCommand = "cd " + shellQuote(r.NextDir) + " && " + r.NextCommand + " " + strings.Join(r.NextArgs, " ")
	}
	return out
}

type addOutput struct {
	Outcome          string                  `json:"outcome,omitempty"`
	ErrorCode        generator.ErrorKind     `json:"error_code,omitempty"`
	ExitCode         int                     `json:"exit_code"`
	Integrations     []string                `json:"integrations,omitempty"`
	Added            []string                `json:"added,omitempty"`
	AlreadyPresent   []string                `json:"already_present,omitempty"`
	FilesCreated     []string                `json:"files_created,omitempty"`
	FilesOverwritten []string                `json:"files_overwritten,omitempty"`
	DryRun           bool                    `json:"dry_run"`
	VerifyRequested  bool                    `json:"verify_requested"`
	VerifyPassed     bool                    `json:"verify_passed"`
	Checks           []generator.CheckResult `json:"checks,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	Error            string                  `json:"error,omitempty"`
}

func addOutputFrom(r generator.AddResult) addOutput {
	return addOutput{Integrations: r.Integrations, Added: r.Added, AlreadyPresent: r.AlreadyPresent,
		FilesCreated: r.FilesCreated, FilesOverwritten: r.FilesOverwritten, DryRun: r.DryRun,
		VerifyRequested: r.VerifyRequested, VerifyPassed: r.VerifyPassed, Checks: r.Checks, Warnings: r.Warnings}
}

func renderCreate(w io.Writer, r generator.CreateResult) {
	for _, warning := range r.Warnings {
		writeOutputf(w, "Warning: %s\n", warning)
	}
	if r.DryRun {
		writeOutputf(w, "Dry run complete for %s (%s)\n", r.ProjectName, r.Mode)
	} else {
		writeOutputf(w, "Created %s (%s)\n", r.ProjectName, r.Mode)
	}
	if r.DryRun && r.Result != nil {
		label := "Applied"
		if r.DryRun {
			label = "Planned"
		}
		writeOutputf(w, "%s: %d create, %d overwrite, %d skip, %d unchanged\n", label, len(r.Result.Created), len(r.Result.Overwritten), len(r.Result.Skipped), len(r.Result.Unchanged))
		for _, group := range []struct {
			label string
			paths []string
		}{{"create", r.Result.Created}, {"overwrite", r.Result.Overwritten}, {"skip", r.Result.Skipped}, {"unchanged", r.Result.Unchanged}} {
			for _, path := range group.paths {
				writeOutputf(w, "  %-10s %s\n", group.label, path)
			}
		}
	}
	if r.VerifyRan && r.VerifyPassed {
		writeOutputln(w, "Verified: tests and build")
	}
	if len(r.NextSteps) > 0 {
		writeOutputln(w, "\nNext:")
		for _, step := range r.NextSteps {
			writeOutputf(w, "  %s\n", step)
		}
	}
}

func writeOutputf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeOutputln(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}

func shellQuote(path string) string { return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'" }
