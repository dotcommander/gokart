package main

import (
	"context"
	"io"
	"os"
	"runtime/debug"

	"github.com/dotcommander/gokart/cmd/gokart/internal/commands"
	"github.com/dotcommander/gokart/cmd/gokart/internal/generator"
)

func run(ctx context.Context, args []string, binaryPath string, stdout, stderr io.Writer) int {
	version := effectiveGokartVersion(gokartVersion)
	projects := generator.New(generator.Dependencies{GeneratorVersion: version, LookupEnv: os.LookupEnv})
	return commands.Execute(ctx, args, version, commands.Dependencies{
		Projects: projects, Stdout: stdout, Stderr: stderr, Getwd: os.Getwd,
		UserConfigDir: os.UserConfigDir, BinaryPath: binaryPath,
	})
}

func effectiveGokartVersion(version string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return selectGokartVersion(version, "")
	}
	return selectGokartVersion(version, info.Main.Version)
}

func selectGokartVersion(version, moduleVersion string) string {
	if version != "dev" || moduleVersion == "" || moduleVersion == "(devel)" {
		return version
	}
	return moduleVersion
}
