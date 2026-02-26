package server

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAnalysisWithTestdata(t *testing.T) {
	// go test sets cwd to the package directory.
	dir := filepath.Join("..", "..", "testdata", "01_single_iface")

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	data, cleanup, err := RunAnalysis(context.Background(), AnalysisConfig{
		Input: dir,
	}, logger)
	t.Cleanup(cleanup)
	require.NoError(t, err)

	assert.NotEmpty(t, data.Interfaces, "expected at least one interface")
	assert.NotEmpty(t, data.Types, "expected at least one type")
	assert.NotEmpty(t, data.Relations, "expected at least one relation")
	assert.Equal(t, dir, data.RepoAddress)
}
