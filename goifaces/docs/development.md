# Development

## Prerequisites

- Go 1.21+
- golangci-lint: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- Node.js (for visual verification only): `npm install -g @mermaid-js/mermaid-cli`

## Setup

```bash
git clone <repo>
cd goifaces
go mod download
git config core.hooksPath .githooks
```

## Build

```bash
go build -o goifaces .
```

## Lint

```bash
golangci-lint run ./...
```

Config: `.golangci.yml` enables errcheck, govet, staticcheck, unused, gosimple, ineffassign, gofmt, misspell.

## Test

```bash
go test ./...
```

Tests are in `internal/integration_test.go` — 10 end-to-end tests using testdata directories.

## Pre-commit Hook

The `.githooks/pre-commit` hook runs both `golangci-lint run ./...` and `go test ./...`. Commits are blocked if either fails.

Configured via: `git config core.hooksPath .githooks`

## Visual Verification

After tests pass:

```bash
bash scripts/visual-verify.sh
```

This renders all testdata Mermaid outputs to SVG using `mmdc` (mermaid-cli). Agents can then read the SVGs to visually verify diagrams.

## Project Structure

```
goifaces/
  main.go                       # CLI entry point
  internal/
    logging/logging.go          # slog JSON handler setup
    resolver/resolver.go        # Input resolution (local/GitHub)
    analyzer/
      types.go                  # Data structures
      analyzer.go               # Package loading + type analysis
      filter.go                 # Filtering logic
    enricher/
      enricher.go               # Enricher interface + types
      grouper.go                # Package grouping
      patterns.go               # Pattern detection (no-op)
      simplifier.go             # Node cap + orphan pruning
      annotator.go              # Annotation (no-op)
      scorer.go                 # Relationship scoring (equal weight)
    diagram/mermaid.go          # Mermaid generation
    server/server.go            # HTTP server + browser
  testdata/                     # Self-contained Go modules for testing
  docs/                         # Project documentation
  scripts/                      # Utility scripts
  logs/                         # Log output (gitignored)
```
