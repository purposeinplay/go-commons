package psqlutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrUnableToResolveCaller is returned when the caller CWD cannot be retrieved.
var ErrUnableToResolveCaller = errors.New("unable to resolve caller")

// ReadSchema reads schema dynamically based on the CWD of the caller.
func ReadSchema(projectDirectoryName string) (string, error) {
	path, err := getDirectoryPath(projectDirectoryName)
	if err != nil {
		return "", fmt.Errorf("get project directory: %w", err)
	}

	schemaPath := filepath.Clean(
		filepath.Join(
			string(os.PathSeparator),
			filepath.Join(
				path,
				"sql",
				"schema.sql",
			),
		),
	)

	schemaB, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("err while reading schema: %w", err)
	}

	return string(schemaB), nil
}

// ErrDirectoryNotFound is returned when the
// project directory is not found.
var ErrDirectoryNotFound = errors.New("wallee directory not found")

func getDirectoryPath(directoryName string) (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", ErrUnableToResolveCaller
	}

	pathParts := strings.Split(filename, string(os.PathSeparator))

	var directoryPath string

	// reverse range over path parts to find the directory
	// absolute path
	for directoryPath == "" && len(pathParts) > 0 {
		if pathParts[len(pathParts)-1] != directoryName {
			pathParts = pathParts[:len(pathParts)-1]

			continue
		}

		directoryPath = filepath.Join(pathParts...)
	}

	if directoryPath == "" {
		return "", ErrDirectoryNotFound
	}

	return directoryPath, nil
}
