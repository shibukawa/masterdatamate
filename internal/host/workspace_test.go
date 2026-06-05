package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspaceFromProjectDescendant(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.test/project\n")
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))
	nested := filepath.Join(root, "docs", "notes")
	mkdirAll(t, nested)

	got, err := ResolveWorkspace(nested)
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveWorkspace() = %q, want %q", got, root)
	}
}

func TestResolveWorkspaceFromInsideMasterdata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), "{}\n")
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations", "0000_initial"))

	got, err := ResolveWorkspace(filepath.Join(root, "masterdata", "generations", "0000_initial"))
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveWorkspace() = %q, want %q", got, root)
	}
}

func TestResolveWorkspaceIgnoresBinaryPathOnlyRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.test/project\n")
	mkdirAll(t, filepath.Join(root, "masterdata", "schema"))
	mkdirAll(t, filepath.Join(root, "masterdata", "generations"))
	dist := filepath.Join(root, "dist-native")
	mkdirAll(t, dist)
	binary := filepath.Join(dist, "masterdatamate")
	writeFile(t, binary, "")

	got, err := ResolveWorkspace(binary)
	if err != nil {
		t.Fatalf("ResolveWorkspace() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveWorkspace() = %q, want %q", got, root)
	}
}

func TestResolveWorkspaceRequiresMasterdataAtMarkedRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.test/project\n")

	_, err := ResolveWorkspace(root)
	if err == nil {
		t.Fatal("ResolveWorkspace() error = nil, want error")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
