#!/bin/bash
# scripts/visual-verify.sh — render all test outputs to SVG for visual inspection
set -e

BINARY=${1:-./goifaces}

if ! command -v mmdc &> /dev/null; then
    echo "Error: mmdc not found. Install with: npm install -g @mermaid-js/mermaid-cli"
    exit 1
fi

if [ ! -f "$BINARY" ]; then
    echo "Error: binary not found at $BINARY. Run 'go build -o goifaces .' first."
    exit 1
fi

for dir in testdata/*/; do
    name=$(basename "$dir")
    echo "Rendering $name..."
    $BINARY -output "$dir/output.mmd" -log-level warn "$dir" 2>/dev/null || true
    if [ -f "$dir/output.mmd" ]; then
        # Skip empty diagrams (just "classDiagram" with no content)
        if [ "$(wc -l < "$dir/output.mmd")" -le 1 ]; then
            echo "  → skipped (empty diagram)"
            rm -f "$dir/output.mmd"
            continue
        fi
        if mmdc -i "$dir/output.mmd" -o "$dir/output.svg" -t default --quiet 2>/dev/null; then
            echo "  → ${dir}output.svg"
        else
            echo "  → skipped (mmdc render failed)"
        fi
    else
        echo "  → skipped (no output)"
    fi
done

echo "Done. Agent can now read any SVG with: Read tool on testdata/NN_xxx/output.svg"
