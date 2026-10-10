---
title: CLI
description: The vanguard command-line interface — scan, check, explain, init, flags, output formats, and the exit-code contract.
---

# Vanguard CLI

`vanguard` is a single static binary. The commands cover the day-to-day
flows: scan a tree, gate CI, explain a rule, and bootstrap a config.

## Commands

| Command | Purpose |
|---|---|
| `vanguard scan <path>` | Full scan with a pretty (TTY), JSON, or SARIF report |
| `vanguard check <path>` | CI alias of `scan` — same pipeline and exit codes; on non-TTY stdout prints one summary line |
| `vanguard explain <rule-id>` | Full documentation for one rule: why, what fires, what stays silent, compliant example |
| `vanguard init` | Generate a starter `.vanguard.yaml` with the full rule-id list |
| `vanguard version` | One line: `vanguard 0.1.0 (commit <sha>, built <date>)` |

## Flags (scan / check)

| Flag | Default | Meaning |
|---|---|---|
| `--format` | `pretty` | Output format: `pretty` (human), `json`, or `sarif` |
| `-o, --output FILE` | *(stdout)* | Write the report to FILE instead of stdout |
| `--config FILE` | *(discovery)* | Explicit config file; skips `.vanguard.yaml` discovery |
| `--no-config` | `false` | Ignore config discovery; run with built-in defaults |
| `--severity` | `INFO` | Display threshold: `INFO`, `WARN`, or `ERROR` (display only — never changes the exit code) |
| `--rules` | *(all)* | Comma-separated rule ids or family prefixes to enable |
| `--no-color` | `false` | Disable ANSI color even on a terminal |
| `-v, --verbose` | `false` | Evidence trail on stderr: walker, adapter pick, skips, suppression notes |
| `--timing` | `false` | Include the engine-measured scan duration in the report (default output is deterministic) |
| `--list-rules` | `false` | Print the sorted rule catalog with full metadata, then exit 0 |

## Exit Codes

| Code | Condition | CI semantics |
|---|---|---|
| `0` | No surviving ERROR finding — clean, WARN/INFO only, or no API surface | pass |
| `1` | ≥ 1 surviving **ERROR**-severity finding (after config + suppression) | fail the job |
| `2` | Tool or config error (bad `.vanguard.yaml`, unknown flag/rule option, unreadable root) | fail the job — operations problem, not a violation |

WARN and INFO findings never change the exit code. The contract is specified
in [Configuration §5](/config) and consumed by
[CI Integration](/ci-integration).

## Output Formats

### pretty (default)

TTY-aware: color, findings grouped by rule and severity, one fix hint per
finding, and a final summary line. Non-TTY invocations (CI logs) render the
plain variant; `check` collapses it to a single line:

```text
vanguard check: findings — 1 ERROR, 1 WARN, 0 INFO, 0 suppressed (exit 1)
```

The verdict word is `clean` on exit 0 and `findings` on exit 1.

### json

A stable machine contract, including:

```json
{
  "summary": {
    "findings": { "error": 1, "warning": 1, "info": 0 },
    "suppressed": 0
  }
}
```

Trend dashboards should ingest `summary.findings.*` and `suppressed`.

### sarif

SARIF 2.1.0, schema-validated, with `partialFingerprints` derived from
(rule, file, line, column, message) — GitHub tracks the same alert across
pushes instead of reopening it. Byte-deterministic across runs.

## Examples

```bash
# First look at a repo
vanguard scan /path/to/project

# Only errors and warnings, JSON to a file
vanguard scan /path/to/project --format json -o vanguard.json --severity WARN

# SARIF for the GitHub Security tab
vanguard scan /path/to/project --format sarif --output vanguard.sarif

# Run one family only
vanguard scan /path/to/project --rules R4xx

# Why does R4xx-02 fire?
vanguard explain R4xx-02

# Bootstrap config
vanguard init
```

## References

- [Installation](/install)
- [Configuration](/config)
- [CI Integration](/ci-integration)
- [Rule Catalog](/rules/README)
