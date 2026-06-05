package host

import (
	"fmt"
	"os"
	"path/filepath"
)

var workspaceRootMarkers = []string{"go.mod", ".git", "package.json"}

func NewWorkspacePath(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("workspace root is empty")
	}
	return filepath.Abs(root)
}

// ResolveWorkspace discovers the workspace root for implicit launches.
// Discovery starts at start and walks upward until a project root marker and
// masterdata directory are found in the same directory.
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
		if hasWorkspaceRootMarker(current) && hasDirectory(filepath.Join(current, "masterdata")) {
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

func hasWorkspaceRootMarker(dir string) bool {
	for _, marker := range workspaceRootMarkers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

func hasDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
