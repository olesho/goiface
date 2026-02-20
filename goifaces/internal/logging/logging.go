package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Setup configures slog to write JSONL to both stderr and a log file.
// Returns a logger and a cleanup function to close the file handle.
func Setup(logFile string, level slog.Level) (*slog.Logger, func(), error) {
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return nil, nil, err
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}

	w := io.MultiWriter(os.Stderr, f)
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)

	cleanup := func() {
		_ = f.Close()
	}

	return logger, cleanup, nil
}
