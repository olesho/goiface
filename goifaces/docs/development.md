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
      grouper.go                # Package grouping (default)
      patterns.go               # Pattern detection (default no-op)
      simplifier.go             # Node cap + orphan pruning (default)
      annotator.go              # Annotation (default no-op)
      scorer.go                 # Relationship scoring (default equal weight)
      llm_grouper.go            # LLM semantic grouper
      llm_patterns.go           # LLM pattern detector
      llm_simplifier.go         # LLM intelligent simplifier
      llm_annotator.go          # LLM annotator
      llm_scorer.go             # LLM relationship scorer
      llm/
        client.go               # OpenAI-compatible HTTP client
        serialize.go            # Result serialization for prompts
    diagram/mermaid.go          # Mermaid generation
    server/server.go            # HTTP server + browser
  testdata/                     # Self-contained Go modules for testing
  docs/                         # Project documentation
  scripts/                      # Utility scripts
  logs/                         # Log output (gitignored)
```
