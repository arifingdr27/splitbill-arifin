package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func ensureLocalDirs(basePath string) error {
	dirs := []string{
		basePath,
		filepath.Join(basePath, "images"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return nil
}
