package psqlutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ReadSchema reads schema dynamically based on the CWD of the caller.
func ReadSchema() (string, error) {
	sqlDir, err := FindSQLDir()
	if err != nil {
		return "", fmt.Errorf("find sql dir: %w", err)
	}

	const schemaFile = "schema.sql"

	schemaB, err := os.ReadFile(filepath.Join(filepath.Clean(sqlDir), schemaFile))
	if err != nil {
		return "", fmt.Errorf("read schema: %w", err)
	}

	return string(schemaB), nil
}

var (
	// ErrUnableToResolveCaller is returned when the caller CWD cannot be retrieved.
	ErrUnableToResolveCaller = errors.New("unable to resolve caller")

	// ErrProjectRootNotFound is returned when the
	// project root is not found.
	ErrProjectRootNotFound = errors.New("project root not found")
)

func getProjectRoot() (string, error) {
	rootIndicators := []string{"go.mod"}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", ErrUnableToResolveCaller
	}

	dir := filepath.Dir(filename)

	// Walk up the directory tree until we find the project root.
	for {
		for _, indicator := range rootIndicators {
			if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
				return dir, nil
			}
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break // we've reached the root of the filesystem and didn't find the project root
		}

		dir = parentDir
	}

	return "", ErrProjectRootNotFound
}
