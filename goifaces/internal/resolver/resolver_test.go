package resolver

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestFindModuleRootRecursive(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, root string) // creates the directory structure
		wantRel string                          // expected result relative to root ("" means root itself)
		wantErr bool
	}{
		{
			name: "go.mod at root",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "go.mod"), "module example\n")
			},
			wantRel: "",
			wantErr: false,
		},
		{
			name: "go.mod in subdirectory",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, filepath.Join(root, "subdir"))
				writeFile(t, filepath.Join(root, "subdir", "go.mod"), "module example\n")
			},
			wantRel: "subdir",
			wantErr: false,
		},
		{
			name: "no go.mod anywhere",
			setup: func(t *testing.T, root string) {
				// empty directory — no go.mod
			},
			wantErr: true,
		},
		{
			name: "skips .git directory",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, filepath.Join(root, ".git"))
				writeFile(t, filepath.Join(root, ".git", "go.mod"), "module fake\n")
				mkdirAll(t, filepath.Join(root, "real"))
				writeFile(t, filepath.Join(root, "real", "go.mod"), "module real\n")
			},
			wantRel: "real",
			wantErr: false,
		},
		{
			name: "picks shallowest go.mod",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, filepath.Join(root, "a"))
				writeFile(t, filepath.Join(root, "a", "go.mod"), "module a\n")
				mkdirAll(t, filepath.Join(root, "a", "b"))
				writeFile(t, filepath.Join(root, "a", "b", "go.mod"), "module a/b\n")
			},
			wantRel: "a",
			wantErr: false,
		},
		{
			name: "multiple go.mod at same depth returns alphabetically first",
			setup: func(t *testing.T, root string) {
				mkdirAll(t, filepath.Join(root, "beta"))
				writeFile(t, filepath.Join(root, "beta", "go.mod"), "module beta\n")
				mkdirAll(t, filepath.Join(root, "alpha"))
				writeFile(t, filepath.Join(root, "alpha", "go.mod"), "module alpha\n")
			},
			wantRel: "alpha",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			got, err := findModuleRootRecursive(root)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result: %s)", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var want string
			if tt.wantRel == "" {
				want = root
			} else {
				want = filepath.Join(root, tt.wantRel)
			}

			if got != want {
				t.Errorf("got %s, want %s", got, want)
			}
		})
	}
}

// mkdirAll is a test helper that creates a directory and all parents.
func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

// writeFile is a test helper that writes content to a file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func TestResolve_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	got, cleanup, err := Resolve(context.Background(), dir, slog.Default())
	defer cleanup()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != dir {
		t.Errorf("got %s, want %s", got, dir)
	}
}

func TestResolve_WithGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example\n\ngo 1.21\n")

	got, cleanup, err := Resolve(context.Background(), dir, slog.Default())
	defer cleanup()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != dir {
		t.Errorf("got %s, want %s", got, dir)
	}
}

func TestResolve_NonExistentPath(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "does-not-exist")

	_, cleanup, err := Resolve(context.Background(), nonexistent, slog.Default())
	defer cleanup()

	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestResolve_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notadir.txt")
	writeFile(t, filePath, "hello")

	_, cleanup, err := Resolve(context.Background(), filePath, slog.Default())
	defer cleanup()

	if err == nil {
		t.Fatal("expected error for file path, got nil")
	}
}
