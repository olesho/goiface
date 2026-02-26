# Architecture

## Overview

goifaces is a Go CLI tool that analyzes Go codebases and produces Mermaid class diagrams showing interface-implementation relationships.

## Data Flow

```
With argument:
  Input (path/URL) → Resolver → Analyzer → Filter → Enricher Pipeline → Diagram Generator → Server/File

Without argument (landing page mode):
  No input → Server (landing page) → User enters path → POST /api/load
           → Resolver → Analyzer → Filter → Enricher Pipeline → Diagram Generator → Interactive UI
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
- `PrepareInteractiveData()` — converts analysis results into `InteractiveData` struct with sanitized IDs, method signatures, and full `PkgPath` for the interactive web UI; the `PkgPath` field on `InteractiveInterface` and `InteractiveType` enables client-side cross-referencing between treemap blocks and their interfaces/types
- `FilterBySelection()` — filters a Result to only include selected items and their direct relations (used for testing the client-side JS filtering logic)
- `NodeID()` / `SanitizeSignature()` — exported utilities for consistent node ID and method signature handling
- `BuildSlides()` — legacy slide generation using a pluggable `Splitter` interface (retained for backward compatibility)

`DiagramOptions.IncludeInit` controls whether the `%%{init:}%%` theme directive is emitted. File output (`-output`) sets this to `true` for standalone `.mmd` rendering; server mode omits it so that `mermaid.initialize()` in the HTML page handles theming — this prevents the init directive from overriding `classDef` custom styles in Mermaid v11.

### `internal/diagram/split`
Slide splitting strategies. Defines the `Splitter` interface and `Group` type.
- **HubAndSpoke** — identifies high-connectivity interfaces (hubs, connections >= threshold) that repeat on every detail slide, then chunks remaining types (spokes) into groups. Non-hub interfaces are attached to the chunk containing their connected types. A post-filter in `subResultForSplitGroup` removes orphaned interfaces and types that have no surviving relations on a given slide.

### `internal/server/analyze.go`
Extracted analysis pipeline shared by CLI and server modes.
- **`AnalysisConfig`** — holds pipeline parameters: `Input`, `Filter`, `IncludeStdlib`, `IncludeUnexported`
- **`RunAnalysis(ctx, cfg, logger)`** — executes the full resolve → analyze → filter → enrich → prepare pipeline; returns `InteractiveData`, a cleanup function, and an error

### `internal/server` (server.go)
HTTP server serving an interactive tabbed HTML UI with embedded Mermaid.js rendering.

**`serverState`** — holds mutable server state protected by a `sync.RWMutex`. Fields: `data *InteractiveData`, `tmpl *template.Template`, `cleanup func()`. When `data` is nil the server renders the landing page; once data is loaded via `/api/load` it switches to the interactive template.

**`ServeInteractiveNoData(ctx, port, openBrowser, logger)`** — starts the HTTP server without pre-loaded analysis data. `GET /` renders the landing page (path input form) when no data is loaded, or the full interactive UI once data has been loaded. Blocks until the context is cancelled.

**`POST /api/load`** — accepts `{"path": "<local-path>"}`, runs the analysis pipeline via `RunAnalysis`, and swaps the server state to the interactive view. Validation rules:
- Request body must be valid JSON (400 otherwise)
- `path` field is required and trimmed (400 if empty)
- HTTP/HTTPS URLs are rejected (400 — use the `-path` CLI flag for GitHub URLs)
- On success returns `{"ok": true}`; on analysis failure returns `{"error": "<message>"}`

Three tabs:
- **Package Map** — native HTML/CSS squarified treemap visualization of the package hierarchy; uses vanilla JS with no external libraries; fills the entire viewport with proportionally-sized rectangles; rendered immediately on page load; clicking a package block with interfaces or types shows a floating overlay listing the package's interfaces and types (click again or click outside to dismiss); client-side lookup maps (`pkgInterfaces`, `pkgTypes`) are built from the `data` JSON at init time, keyed by `pkgPath`
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
