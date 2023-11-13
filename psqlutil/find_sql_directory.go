package psqlutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FindSQLDir attempts to compose the path to the sql directory in the project.
// nolint: gocognit // allow high cog complexity.
func FindSQLDir() (string, error) {
	projectRoot, err := getProjectRoot()
	if err != nil {
		return "", fmt.Errorf("get project root: %w", err)
	}

	var sqlDirPath string

	if err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err // Propagate any error encountered.
		}

		if d.IsDir() {
			// Check for 'schema.sql' inside this directory.
			schemaFilePath := filepath.Join(path, "schema.sql")

			if _, err := os.Stat(schemaFilePath); err != nil {
				if os.IsNotExist(err) {
					return nil // File not found, continue walking.
				}

				return fmt.Errorf("stat %q: %w", schemaFilePath, err)
			}

			sqlDirPath = path // Found the directory containing 'schema.sql'.

			return fs.SkipAll // Throw an error to stop the walk early
		}

		return nil
	}); err != nil {
		return "", fmt.Errorf("walk project root: %w", err)
	}

	if sqlDirPath == "" {
		return "", fs.ErrNotExist
	}

	return sqlDirPath, nil
}
