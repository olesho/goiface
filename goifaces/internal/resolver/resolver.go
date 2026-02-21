package resolver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

	// Find module root (nearest go.mod)
	modRoot, err := findModuleRoot(absPath)
	if err != nil {
		return "", cleanup, err
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

// findModuleRootRecursive searches dir and its subdirectories for a go.mod file,
// returning the directory containing the first one found (breadth-first).
func findModuleRootRecursive(root string) (string, error) {
	// Check root first
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return root, nil
	}

	// Search subdirectories (breadth-first, max depth 3)
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		sub := filepath.Join(root, e.Name())
		if _, err := os.Stat(filepath.Join(sub, "go.mod")); err == nil {
			return sub, nil
		}
	}

	return "", fmt.Errorf("no go.mod found in %s or immediate subdirectories", root)
}

func goModDownload(ctx context.Context, dir string, logger *slog.Logger) error {
	logger.Debug("running go mod download", "dir", dir)
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
