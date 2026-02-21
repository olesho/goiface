# CLI Reference

## Usage

```
goifaces [flags] <path-or-url>
```

The first positional argument is the Go code to analyze. Can be:
- Local directory: `./my-project`
- Sub-package: `./my-project/internal/auth`
- GitHub URL: `https://github.com/user/repo`

## Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `-path` | string | (positional arg) | Alternative to positional argument for input path/URL |
| `-port` | int | `8080` | HTTP server port |
| `-filter` | string | (none) | Package path prefix filter — only show matching packages |
| `-include-stdlib` | bool | `false` | Include stdlib interface implementations (io.Reader, fmt.Stringer, error, etc.) |
| `-include-unexported` | bool | `false` | Include unexported interfaces and types |
| `-output` | string | (none) | Write Mermaid to file instead of starting HTTP server |
| `-no-browser` | bool | `false` | Don't auto-open browser when starting server |
| `-log-file` | string | `logs/goifaces.log` | Path to JSONL log file |
| `-log-level` | string | `info` | Log level: debug, info, warn, error |
| `-slide-threshold` | int | `20` | Node count above which slide mode activates |
| `-hub-threshold` | int | `3` | Min connections for an interface to be a hub (repeated on every slide) |
| `-chunk-size` | int | `3` | Max implementations per detail slide |
| `-enrich` | bool | `false` | Enable LLM-backed enrichment (requires `GOIFACES_LLM_API_KEY`) |

## Examples

```bash
# Analyze a local project, open in browser
goifaces ./my-project

# Analyze a specific package
goifaces ./my-project/internal/auth

# Analyze a GitHub repo
goifaces https://github.com/hashicorp/go-memdb

# Save diagram to file
goifaces ./my-project -output diagram.md

# Include stdlib interfaces
goifaces ./my-project -include-stdlib

# Debug logging
goifaces ./my-project -log-level debug

# Filter to specific packages
goifaces ./my-project -filter github.com/user/repo/internal

# Control slide splitting for large diagrams
goifaces https://github.com/hashicorp/go-memdb -hub-threshold 3 -chunk-size 4

# Enable LLM enrichment (semantic grouping, pattern detection)
GOIFACES_LLM_API_KEY=sk-... goifaces ./my-project -enrich

# Use a custom LLM endpoint (e.g., Ollama)
GOIFACES_LLM_ENDPOINT=http://localhost:11434/v1 GOIFACES_LLM_MODEL=llama3 goifaces ./my-project -enrich
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `GOIFACES_LLM_API_KEY` | (none) | API key for the LLM endpoint (required when `--enrich` is set) |
| `GOIFACES_LLM_ENDPOINT` | `https://api.openai.com/v1` | Base URL for OpenAI-compatible API |
| `GOIFACES_LLM_MODEL` | `gpt-4o-mini` | Model identifier to use for enrichment |
