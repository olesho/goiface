# Agent Workflow

## Build
go build -o goifaces .

## Run (with debug logs for full visibility)
./goifaces ./path/to/target -log-level debug -log-file logs/goifaces.log

## Read logs (JSONL — each line is a JSON object)
# Last 20 log lines:
tail -20 logs/goifaces.log

# Errors only:
grep '"level":"ERROR"' logs/goifaces.log

# Warnings and errors:
grep -E '"level":"(ERROR|WARN)"' logs/goifaces.log

# Filter by component:
grep '"component":"analyzer"' logs/goifaces.log

# Pretty-print a specific line:
tail -1 logs/goifaces.log | jq .

## Lint (must pass before committing)
golangci-lint run ./...

## Test
go test ./...

## Quick feedback loop
go build -o goifaces . && ./goifaces ./testdata/01_single_iface -output /dev/stdout -log-level debug -log-file logs/goifaces.log 2>/dev/null; echo "---LOGS---"; tail -30 logs/goifaces.log

## Visual Verification
After tests pass, render diagrams and visually inspect:

# 1. Render all test outputs to SVG
bash scripts/visual-verify.sh

# 2. Read a specific rendered diagram (Claude Code will see the image)
# Use the Read tool on: testdata/01_single_iface/output.svg

# 3. Read the corresponding input Go code
# Use the Read tool on: testdata/01_single_iface/shapes.go

# 4. Verify: does the diagram correctly show the interfaces,
#    implementations, and relationships from the Go code?

# If mismatch found: file a bug describing what's wrong
