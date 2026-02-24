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
Generates Mermaid `classDiagram` syntax from analysis results. Uses `direction LR` layout so implementations appear on the left and interfaces on the right. Interface blocks (blue) display `<<interface>>` tag and method signatures; implementation blocks (green) show only the type name -- methods are omitted from impl blocks because they are already listed in the interface blocks, reducing visual clutter. Handles node ID sanitization, method truncation, deterministic ordering.

Key exported functions:
- `GenerateMermaid()` — full class diagram from analysis results
- `GeneratePackageMapMermaid()` — flowchart showing repository package hierarchy with per-package interface/type counts; each package node gets a distinct pastel background color from a fixed palette
- `PreparePackageMapData()` — converts analysis results into a `[]*PackageMapNode` tree for client-side HTML treemap rendering; reuses the same tree-building logic as `GeneratePackageMapMermaid`
- `PrepareInteractiveData()` — converts analysis results into `InteractiveData` struct with sanitized IDs and method signatures for the interactive web UI
- `FilterBySelection()` — filters a Result to only include selected items and their direct relations (used for testing the client-side JS filtering logic)
- `NodeID()` / `SanitizeSignature()` — exported utilities for consistent node ID and method signature handling
- `BuildSlides()` — legacy slide generation using a pluggable `Splitter` interface (retained for backward compatibility)

`DiagramOptions.IncludeInit` controls whether the `%%{init:}%%` theme directive is emitted. File output (`-output`) sets this to `true` for standalone `.mmd` rendering; server mode omits it so that `mermaid.initialize()` in the HTML page handles theming — this prevents the init directive from overriding `classDef` custom styles in Mermaid v11.

### `internal/diagram/split`
Slide splitting strategies. Defines the `Splitter` interface and `Group` type.
- **HubAndSpoke** — identifies high-connectivity interfaces (hubs, connections >= threshold) that repeat on every detail slide, then chunks remaining types (spokes) into groups. Non-hub interfaces are attached to the chunk containing their connected types. A post-filter in `subResultForSplitGroup` removes orphaned interfaces and types that have no surviving relations on a given slide.

### `internal/server`
HTTP server serving an interactive tabbed HTML UI with embedded Mermaid.js rendering. Three tabs:
- **Package Map** — native HTML/CSS squarified treemap visualization of the package hierarchy; uses vanilla JS with no external libraries; fills the entire viewport with proportionally-sized rectangles; rendered immediately on page load
- **Implementations** — scrollable checkbox list of all implementation types; selecting items dynamically generates a Mermaid class diagram showing only selected items and their direct relations
- **Interfaces** — scrollable checkbox list of all interfaces with the same filtering behavior

Selections from both lists are combined (union). Client-side JavaScript handles filtering and Mermaid diagram generation based on checkbox selections. Includes zoom controls, copy-source button, and auto-browser-open.

## Dependencies

| Package | Purpose |
|---|---|
| `golang.org/x/tools/go/packages` | Load and type-check Go packages |
| `go/types` (stdlib) | Interface satisfaction checking |
| `log/slog` (stdlib) | Structured JSON logging |
| `github.com/stretchr/testify` | Test assertions |
| Mermaid.js CDN | Client-side diagram rendering |
