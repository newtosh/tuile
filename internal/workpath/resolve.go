package workpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve validates a workspace path from an authenticated operator (CLI or
// bootstrap API) and returns its absolute directory path.
func Resolve(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("workspace path is required")
	}
	if strings.Contains(raw, "\x00") {
		return "", fmt.Errorf("invalid workspace path")
	}

	cleaned := filepath.Clean(raw)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("workspace path: %w", err)
	}

	if err := assertWithinFilesystemRoot(abs); err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("workspace not found: %w", err)
	}
	if err := assertWithinFilesystemRoot(resolved); err != nil {
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("workspace not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace is not a directory: %s", resolved)
	}
	return resolved, nil
}

func assertWithinFilesystemRoot(path string) error {
	root := filesystemRoot(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("workspace path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("workspace path escapes filesystem root")
	}
	return nil
}

func filesystemRoot(path string) string {
	if vol := filepath.VolumeName(path); vol != "" {
		return vol + string(os.PathSeparator)
	}
	return string(os.PathSeparator)
}
