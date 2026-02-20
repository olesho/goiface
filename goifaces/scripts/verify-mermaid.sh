#!/bin/bash
# scripts/verify-mermaid.sh — verify generated Mermaid syntax is valid
# Usage: ./scripts/verify-mermaid.sh [mermaid-file-or-url]
#
# Accepts either:
#   - A .mmd file path
#   - An HTTP URL to fetch Mermaid source from (e.g. http://localhost:8083/mermaid.md)
#
# Validates by rendering with mmdc (Mermaid CLI). Exit code 0 = valid, 1 = invalid.
set -e

if ! command -v mmdc &> /dev/null; then
    echo "Error: mmdc not found. Install with: npm install -g @mermaid-js/mermaid-cli"
    exit 1
fi

INPUT="${1:?Usage: verify-mermaid.sh <file.mmd or http://url>}"
TMPFILE=""

cleanup() {
    [ -n "$TMPFILE" ] && rm -f "$TMPFILE" "$TMPFILE.svg"
}
trap cleanup EXIT

if [[ "$INPUT" == http://* ]] || [[ "$INPUT" == https://* ]]; then
    TMPFILE=$(mktemp /tmp/verify-mermaid-XXXXXX.mmd)
    echo "Fetching Mermaid source from $INPUT..."
    curl -sf "$INPUT" > "$TMPFILE"
    if [ ! -s "$TMPFILE" ]; then
        echo "FAIL: empty response from $INPUT"
        exit 1
    fi
    MMD_FILE="$TMPFILE"
else
    MMD_FILE="$INPUT"
    if [ ! -f "$MMD_FILE" ]; then
        echo "FAIL: file not found: $MMD_FILE"
        exit 1
    fi
fi

echo "Validating Mermaid syntax..."
echo "  File: $MMD_FILE ($(wc -l < "$MMD_FILE") lines)"

# Render to SVG as validation — mmdc will fail on syntax errors
SVG_OUT="${TMPFILE:-/tmp/verify-mermaid-output}.svg"
if mmdc -i "$MMD_FILE" -o "$SVG_OUT" -t default --quiet 2>&1; then
    echo "PASS: Mermaid syntax is valid"
    exit 0
else
    echo "FAIL: Mermaid syntax error detected"
    echo ""
    echo "First 20 lines of input:"
    head -20 "$MMD_FILE"
    exit 1
fi
