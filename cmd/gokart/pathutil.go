package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func shellQuote(path string) string {
	if path == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

func normalizeProjectArg(projectArg string) (projectName, targetDir string, err error) {
	raw := strings.TrimSpace(projectArg)
	if raw == "" {
		return "", "", errors.New("project name is required")
	}

	cleanArg := filepath.Clean(raw)
	projectName = filepath.Base(cleanArg)

	if projectName == "." || projectName == ".." || projectName == string(filepath.Separator) || projectName == "" {
		return "", "", fmt.Errorf("invalid project name %q", projectArg)
	}

	if !projectNamePattern.MatchString(projectName) {
		return "", "", fmt.Errorf("invalid project name %q (allowed: letters, numbers, ., _, -)", projectName)
	}

	if filepath.IsAbs(raw) {
		targetDir = cleanArg
	} else {
		targetDir = filepath.Join(".", cleanArg)
	}

	return projectName, targetDir, nil
}

func validateModulePath(module string) error {
	mod := strings.TrimSpace(module)
	if mod == "" {
		return errors.New("cannot be empty")
	}

	if strings.ContainsAny(mod, " \t\r\n") {
		return errors.New("cannot contain whitespace")
	}

	if strings.HasPrefix(mod, "/") || strings.HasSuffix(mod, "/") {
		return errors.New("cannot start or end with '/'")
	}

	for _, segment := range strings.Split(mod, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("contains invalid path segment %q", segment)
		}
		if !moduleSegmentPattern.MatchString(segment) {
			return fmt.Errorf("contains invalid path segment %q", segment)
		}
	}

	return nil
}
