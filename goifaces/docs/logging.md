# Logging

## Format

All log output is JSONL (JSON Lines) — one self-contained JSON object per line.

## Output Destinations

- **stderr** — for human visibility during runs
- **Log file** — for agent consumption and post-mortem analysis (default: `logs/goifaces.log`)

Both outputs use `slog.NewJSONHandler`.

## Standard Fields

Every log record contains:

| Field | Type | Description |
|---|---|---|
| `time` | string | ISO 8601 timestamp |
| `level` | string | DEBUG, INFO, WARN, ERROR |
| `msg` | string | Human-readable message |
| `component` | string | Subsystem: resolver, analyzer, diagram, server |

## Log Levels

| Level | Usage |
|---|---|
| DEBUG | Verbose internals: each type checked, each package loaded |
| INFO | Progress milestones: packages loaded, N relations found, server started |
| WARN | Partial failures: package load errors, skipped packages, no go.mod found (local paths) |
| ERROR | Fatal failures: clone failed, no go.mod found in cloned repo |

## Example Log Lines

```json
{"time":"2026-02-19T10:30:00Z","level":"INFO","msg":"packages loaded","component":"analyzer","packages_count":47}
{"time":"2026-02-19T10:30:01Z","level":"DEBUG","msg":"found interface","component":"analyzer","name":"Reader","package":"io"}
{"time":"2026-02-19T10:30:02Z","level":"WARN","msg":"package load error","component":"analyzer","package":"broken/pkg","error":"missing import"}
{"time":"2026-02-19T10:30:03Z","level":"INFO","msg":"analysis complete","component":"analyzer","relations":142}
```

## Reading Logs

```bash
# Last 20 lines
tail -20 logs/goifaces.log

# Errors only
grep '"level":"ERROR"' logs/goifaces.log

# Warnings and errors
grep -E '"level":"(ERROR|WARN)"' logs/goifaces.log

# Filter by component
grep '"component":"analyzer"' logs/goifaces.log

# Pretty-print last line
tail -1 logs/goifaces.log | jq .

# Count by level
jq -r .level logs/goifaces.log | sort | uniq -c
```
