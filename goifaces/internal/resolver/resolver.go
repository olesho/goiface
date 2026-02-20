package resolver

import (
	"context"
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
		return cloneRepo(ctx, input, logger)
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

func cloneRepo(ctx context.Context, url string, logger *slog.Logger) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "goifaces-clone-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("creating temp dir: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	logger.Info("cloning repository", "url", url, "dest", tmpDir)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", url, tmpDir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("git clone: %w", err)
	}

	logger.Info("clone complete", "dest", tmpDir)

	if err := goModDownload(ctx, tmpDir, logger); err != nil {
		logger.Warn("go mod download failed", "error", err)
	}

	return tmpDir, cleanup, nil
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

func goModDownload(ctx context.Context, dir string, logger *slog.Logger) error {
	logger.Debug("running go mod download", "dir", dir)
	cmd := exec.CommandContext(ctx, "go", "mod", "download")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
