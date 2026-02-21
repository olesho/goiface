package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindModuleRootInTree_AtRoot(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\n"), 0o644))

	got, err := findModuleRootInTree(tmp)
	require.NoError(t, err)
	assert.Equal(t, tmp, got)
}

func TestFindModuleRootInTree_InSubdirectory(t *testing.T) {
	tmp := t.TempDir()
	subdir := filepath.Join(tmp, "backend")
	require.NoError(t, os.MkdirAll(subdir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "go.mod"), []byte("module test/backend\n"), 0o644))

	got, err := findModuleRootInTree(tmp)
	require.NoError(t, err)
	assert.Equal(t, subdir, got)
}

func TestFindModuleRootInTree_NoGoMod(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "src"), 0o755))

	_, err := findModuleRootInTree(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no go.mod found")
}

func TestFindModuleRootInTree_SkipsGitDir(t *testing.T) {
	tmp := t.TempDir()
	// Put go.mod in .git (should be skipped)
	gitDir := filepath.Join(tmp, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "go.mod"), []byte("module fake\n"), 0o644))

	// Put go.mod in a real directory
	realDir := filepath.Join(tmp, "real")
	require.NoError(t, os.MkdirAll(realDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "go.mod"), []byte("module real\n"), 0o644))

	got, err := findModuleRootInTree(tmp)
	require.NoError(t, err)
	assert.Equal(t, realDir, got)
}

func TestFindModuleRootInTree_PicksShallowest(t *testing.T) {
	tmp := t.TempDir()

	// Deeper go.mod at a/b/
	deep := filepath.Join(tmp, "a", "b")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "go.mod"), []byte("module deep\n"), 0o644))

	// Shallower go.mod at a/
	shallow := filepath.Join(tmp, "a")
	require.NoError(t, os.WriteFile(filepath.Join(shallow, "go.mod"), []byte("module shallow\n"), 0o644))

	got, err := findModuleRootInTree(tmp)
	require.NoError(t, err)
	assert.Equal(t, shallow, got)
}

func TestFindModuleRootInTree_SameDepthSorted(t *testing.T) {
	tmp := t.TempDir()

	// Two go.mod files at the same depth
	dirA := filepath.Join(tmp, "alpha")
	dirB := filepath.Join(tmp, "beta")
	require.NoError(t, os.MkdirAll(dirA, 0o755))
	require.NoError(t, os.MkdirAll(dirB, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirA, "go.mod"), []byte("module alpha\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dirB, "go.mod"), []byte("module beta\n"), 0o644))

	got, err := findModuleRootInTree(tmp)
	require.NoError(t, err)
	// Should pick alphabetically first
	assert.Equal(t, dirA, got)
}

func TestFindModuleRootInTree_SkipsVendorAndNodeModules(t *testing.T) {
	tmp := t.TempDir()

	// go.mod only in vendor/ and node_modules/ (both should be skipped)
	for _, skip := range []string{"vendor", "node_modules"} {
		d := filepath.Join(tmp, skip)
		require.NoError(t, os.MkdirAll(d, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(d, "go.mod"), []byte("module skip\n"), 0o644))
	}

	_, err := findModuleRootInTree(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no go.mod found")
}
