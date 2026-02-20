# goifaces

Go CLI tool that analyzes Go repositories and visualizes interface-implementation relationships as Mermaid diagrams.

## Mandatory Rules

### 1. Linting
- All code MUST pass `golangci-lint run ./...` before committing
- Pre-commit hook enforces this — do NOT bypass with --no-verify
- Config: `.golangci.yml`

### 2. Structured Logging
- ALL log output MUST use `log/slog` with JSON handler — no `fmt.Println` or `log.Printf` for operational output
- Every log call MUST include a `component` field via the logger passed to each function
- Log levels: DEBUG (verbose), INFO (milestones), WARN (partial failures), ERROR (fatal)
- Log file: `logs/goifaces.log` (JSONL format, one JSON object per line)

### 3. Tests
- ALL tests MUST pass (`go test ./...`) before committing
- Pre-commit hook enforces this — do NOT bypass with --no-verify
- If tests fail, fix the code until they pass. Do NOT skip or delete tests.
- When adding new functionality, add or update tests in `internal/integration_test.go`
- Test workflow: define expected Mermaid → write mock Go code → assert analyzer produces expected output

### 4. Visual Verification
- After tests pass, run `bash scripts/visual-verify.sh` to render diagrams to SVG
- Use the Read tool on `testdata/NN_xxx/output.svg` to visually inspect rendered diagrams
- Compare against the input Go code in the same testdata directory
- If a diagram looks wrong (missing nodes, wrong relationships, layout issues), fix the code
- Prerequisite: `npm install -g @mermaid-js/mermaid-cli` (provides `mmdc`)

### 5. Documentation
- Documentation lives in `docs/` — update it with EVERY code change
- When adding/modifying a package, update `docs/architecture.md`
- When adding/modifying CLI flags, update `docs/cli-reference.md`
- When changing logging behavior, update `docs/logging.md`
- When changing dev setup (linter rules, hooks, build), update `docs/development.md`
- Documentation updates are part of the definition of done — a PR is not complete without them

## Project Documentation
- `docs/architecture.md` — system architecture, package layout, data flow
- `docs/cli-reference.md` — all CLI flags with descriptions and examples
- `docs/logging.md` — JSONL log format, fields, levels, agent log reading patterns
- `docs/development.md` — dev environment setup, linting, git hooks, build, test
