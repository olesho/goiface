package resolver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Resolve takes an input (local dir, sub-package path, or GitHub URL) and returns
// a local directory ready for analysis, plus a cleanup function.
func Resolve(ctx context.Context, input string, logger *slog.Logger) (dir string, cleanup func(), err error) {
	cleanup = func() {} // default no-op

	if isGitHubURL(input) {
		return fetchRepo(ctx, input, logger)
	}

	// Local path
	absPath, err := filepath.Abs(input)
	if err != nil {
		return "", cleanup, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", cleanup, fmt.Errorf("stat %s: %w", absPath, err)
	}

	if !info.IsDir() {
		return "", cleanup, fmt.Errorf("%s is not a directory", absPath)
	}

	// Find module root (nearest go.mod) — optional
	modRoot, err := findModuleRoot(absPath)
	if err != nil {
		// No go.mod found — use the input directory directly.
		// Go analysis will be attempted but may produce empty results.
		logger.Warn("no go.mod found, using directory as-is", "dir", absPath)
		return absPath, cleanup, nil
	}

	logger.Info("resolved local directory", "input", input, "module_root", modRoot)

	// Run go mod download to ensure deps are available
	if err := goModDownload(ctx, modRoot, logger); err != nil {
		logger.Warn("go mod download failed", "error", err)
	}

	return modRoot, cleanup, nil
}

func isGitHubURL(input string) bool {
	return strings.Contains(input, "github.com") &&
		(strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://"))
}

// cacheDir returns a stable directory for caching a cloned repo.
// Uses ~/.cache/goifaces/repos/<hash> where hash is derived from the URL.
func cacheDir(url string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	h := sha256.Sum256([]byte(url))
	name := fmt.Sprintf("%x", h[:8])
	return filepath.Join(home, ".cache", "goifaces", "repos", name), nil
}

// fetchRepo either pulls an existing cached clone or does a fresh clone.
// Returns the module root directory and a no-op cleanup (cache is persistent).
func fetchRepo(ctx context.Context, url string, logger *slog.Logger) (string, func(), error) {
	noop := func() {}

	dir, err := cacheDir(url)
	if err != nil {
		return "", noop, err
	}

	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Cached clone exists — pull latest
		logger.Info("updating cached repository", "url", url, "dir", dir)
		cmd := exec.CommandContext(ctx, "git", "fetch", "--depth=1", "origin")
		cmd.Dir = dir
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			logger.Warn("git fetch failed, will re-clone", "error", err)
			_ = os.RemoveAll(dir)
			return cloneRepo(ctx, url, dir, logger)
		}
		// Reset to fetched HEAD
		cmd = exec.CommandContext(ctx, "git", "reset", "--hard", "origin/HEAD")
		cmd.Dir = dir
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			logger.Warn("git reset failed, will re-clone", "error", err)
			_ = os.RemoveAll(dir)
			return cloneRepo(ctx, url, dir, logger)
		}
		logger.Info("repository updated", "dir", dir)
	} else {
		// Fresh clone
		return cloneRepo(ctx, url, dir, logger)
	}

	// Find module root
	modRoot, err := findModuleRootRecursive(dir)
	if err != nil {
		return "", noop, fmt.Errorf("no go.mod found in cached repo: %w", err)
	}

	logger.Info("found module root", "module_root", modRoot)

	if err := goModDownload(ctx, modRoot, logger); err != nil {
		logger.Warn("go mod download failed", "error", err)
	}

	return modRoot, noop, nil
}

func cloneRepo(ctx context.Context, url, dir string, logger *slog.Logger) (string, func(), error) {
	noop := func() {}

	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", noop, fmt.Errorf("creating cache dir: %w", err)
	}

	logger.Info("cloning repository", "url", url, "dest", dir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", url, dir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		return "", noop, fmt.Errorf("git clone: %w", err)
	}

	logger.Info("clone complete", "dest", dir)

	// Find module root — go.mod may not be at the repo root
	modRoot, err := findModuleRootRecursive(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", noop, fmt.Errorf("no go.mod found in cloned repo: %w", err)
	}

	logger.Info("found module root", "module_root", modRoot)

	if err := goModDownload(ctx, modRoot, logger); err != nil {
		logger.Warn("go mod download failed", "error", err)
	}

	return modRoot, noop, nil
}

func findModuleRoot(dir string) (string, error) {
	current := dir
	for {
		goMod := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no go.mod found in %s or any parent directory", dir)
		}
		current = parent
	}
}

// findModuleRootRecursive searches downward from root for the shallowest go.mod file.
// This is used for cloned repos where go.mod may be in a subdirectory.
func findModuleRootRecursive(root string) (string, error) {
	// Check root first (most common case)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return root, nil
	}

	// BFS through subdirectories to find the shallowest go.mod
	queue := []string{root}
	for len(queue) > 0 {
		var nextLevel []string
		var candidates []string

		for _, dir := range queue {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				name := entry.Name()
				if name == ".git" || name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") {
					continue
				}
				subdir := filepath.Join(dir, name)
				if _, err := os.Stat(filepath.Join(subdir, "go.mod")); err == nil {
					candidates = append(candidates, subdir)
				} else {
					nextLevel = append(nextLevel, subdir)
				}
			}
		}

		if len(candidates) > 0 {
			sort.Strings(candidates)
			return candidates[0], nil
		}
		queue = nextLevel
	}

	return "", fmt.Errorf("no go.mod found in %s or any subdirectory", root)
}

func goModDownload(ctx context.Context, dir string, logger *slog.Logger) error {
	logger.Debug("running go mod download", "dir", dir)
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
