# CLI Reference

## Usage

```
goifaces [flags] [path-or-url]
```

The positional argument is optional. Can be:
- **Omitted:** starts the server with a landing page where you enter a path in the browser
- **Local directory:** `./my-project`
- **Sub-package:** `./my-project/internal/auth`
- **GitHub URL:** `https://github.com/user/repo`

### No-argument (landing page) mode

When no path or URL is given, goifaces starts the HTTP server and opens a landing page in the browser. The landing page presents a text input and "Analyze" button. Enter a local filesystem path to a Go repository and click Analyze — the server runs the analysis pipeline and reloads the page with the full interactive UI (Package Map, Implementations, Interfaces tabs).

The `-output` flag still requires a positional path argument; using `-output` without a path prints an error.

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
| `-enrich` | bool | `false` | Enable LLM-backed enrichment (semantic grouping, pattern detection, intelligent simplification) |

### Environment Variables (for `-enrich`)

| Variable | Default | Description |
|---|---|---|
| `GOIFACES_LLM_API_KEY` | (required) | API key for the OpenAI-compatible endpoint |
| `GOIFACES_LLM_ENDPOINT` | `https://api.openai.com/v1` | API base URL (works with any OpenAI-compatible endpoint) |
| `GOIFACES_LLM_MODEL` | `gpt-4o-mini` | Model identifier |

## Examples

```bash
# Start with landing page (no argument)
goifaces

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

# Enable LLM enrichment (requires API key)
GOIFACES_LLM_API_KEY=sk-... goifaces ./my-project -enrich

# Use a custom OpenAI-compatible endpoint
GOIFACES_LLM_ENDPOINT=http://localhost:11434/v1 GOIFACES_LLM_MODEL=llama3 GOIFACES_LLM_API_KEY=none goifaces ./my-project -enrich
```
