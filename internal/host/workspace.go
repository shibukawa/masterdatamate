package host

import (
	"fmt"
	"os"
	"path/filepath"
)

func NewWorkspacePath(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("workspace root is empty")
	}
	return filepath.Abs(root)
}

// ResolveWorkspace discovers the workspace root for implicit launches.
// Discovery starts at start and selects the nearest directory that contains a
// masterdata directory. Project root markers are diagnostic hints only; they
// must not cause a parent repository to win over a nested sample workspace.
func ResolveWorkspace(start string) (string, error) {
	if start == "" {
		start = "."
	}
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(current)
	if err != nil {
		return "", fmt.Errorf("cannot inspect workspace start %s: %w", current, err)
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		if hasDirectory(filepath.Join(current, "masterdata")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("could not find workspace root containing masterdata from %s", start)
}

func hasDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
