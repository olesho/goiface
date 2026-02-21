# Architecture

## Overview

goifaces is a Go CLI tool that analyzes Go codebases and produces Mermaid class diagrams showing interface-implementation relationships.

## Data Flow

```
Input (path/URL) → Resolver → Analyzer → Filter → Enricher Pipeline → Diagram Generator → Server/File
```

## Package Layout

### `main.go`
CLI entry point. Parses flags, orchestrates the pipeline, handles signals.

### `internal/logging`
Configures `log/slog` with JSON handler for dual output (stderr + log file). Every log line is a self-contained JSON object (JSONL format).

### `internal/resolver`
Resolves input to a local directory:
- Local directory: use as-is
- GitHub URL: `git clone --depth=1` to temp dir
- Finds module root (`go.mod`), runs `go mod download`

### `internal/analyzer`
Core analysis engine:
- **Phase 1:** Load packages via `golang.org/x/tools/go/packages`
- **Phase 2:** Collect interfaces and named types from package scopes
- **Phase 3:** Match implementations using `types.Implements()` with `typeutil.MethodSetCache`

Key types: `InterfaceDef`, `TypeDef`, `MethodSig`, `Relation`, `Result`

### `internal/analyzer` (filter)
Filters results by:
- Stdlib exclusion (default: excluded)
- Unexported exclusion (default: excluded)
- Package path prefix
- Orphan pruning (types/interfaces with no relations)

### `internal/enricher`
Composable pipeline of enrichers. Each implements `Enricher` interface.
- **Grouper** — groups by package (default), or by architectural layer (LLM)
- **Simplifier** — prunes orphans, caps node count by edge rank (default) or architectural significance (LLM)
- **PatternDetector** — detects GoF and Go-specific design patterns (LLM), no-op default
- **Annotator** — generates human-readable descriptions (LLM), no-op default
- **Scorer** — ranks relationships by architectural importance (LLM), equal weight default

Each LLM enricher wraps a default enricher and falls back to it on any error (timeout, malformed response, API failure). Enable with `--enrich` flag.

### `internal/enricher/llm`
Lightweight LLM client abstraction speaking the OpenAI-compatible chat completions API. Uses stdlib `net/http` + `encoding/json` (no external SDK). Features:
- JSON mode (`response_format: {type: "json_object"}`)
- Retry on 5xx (1 retry with backoff)
- Respect `Retry-After` header on 429
- Response body size limit (10 MB)
- API key masking in logs via `slog.LogValuer`
- Result serialization helpers for compact LLM prompts

### `internal/diagram`
Generates Mermaid `classDiagram` syntax from analysis results. Uses `direction LR` layout so implementations appear on the left and interfaces on the right. Interface blocks (blue) display `<<interface>>` tag and method signatures; implementation blocks (green) show only the type name -- methods are omitted from impl blocks because they are already listed in the interface blocks, reducing visual clutter. Handles node ID sanitization, method truncation, deterministic ordering. Slide generation uses a pluggable `Splitter` interface to group nodes into slides. The first slide is a Package Map flowchart showing the repository's package hierarchy with per-package interface/type counts; each package node label displays the module-relative path (e.g. `internal/http/middleware/auth`) rather than just the short package name, while subgraph container titles use only the short segment name for clean grouping. Each package node gets a distinct pastel background color from a fixed palette for visual differentiation. Detail slides show full interface+implementation diagrams with implementation arrows (`..|>`).

`DiagramOptions.IncludeInit` controls whether the `%%{init:}%%` theme directive is emitted. File output (`-output`) sets this to `true` for standalone `.mmd` rendering; server mode omits it so that `mermaid.initialize()` in the HTML page handles theming — this prevents the init directive from overriding `classDef` custom styles in Mermaid v11.

### `internal/diagram/split`
Slide splitting strategies. Defines the `Splitter` interface and `Group` type.
- **HubAndSpoke** — identifies high-connectivity interfaces (hubs, connections >= threshold) that repeat on every detail slide, then chunks remaining types (spokes) into groups. Non-hub interfaces are attached to the chunk containing their connected types. A post-filter in `subResultForSplitGroup` removes orphaned interfaces and types that have no surviving relations on a given slide.

### `internal/server`
HTTP server serving HTML with embedded Mermaid.js rendering. Includes zoom controls, copy button, and auto-browser-open.

## Dependencies

| Package | Purpose |
|---|---|
| `golang.org/x/tools/go/packages` | Load and type-check Go packages |
| `go/types` (stdlib) | Interface satisfaction checking |
| `log/slog` (stdlib) | Structured JSON logging |
| `github.com/stretchr/testify` | Test assertions |
| Mermaid.js CDN | Client-side diagram rendering |
