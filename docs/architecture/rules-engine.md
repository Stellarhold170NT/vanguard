---
title: Rules Engine & Determinism
description: The data-driven rule catalog, the severity model and exit-code contract, the suppression pipeline, and Vanguard's determinism and performance guarantees.
---

# Rules Engine & Determinism

Stage 3–4 of the pipeline: how rules are registered and evaluated, how
severity maps to exit codes, how suppressions apply, and what the engine
guarantees about reproducibility.

## 1. A Data-Driven Catalog

Every rule is one YAML metadata file in `rules/data/` plus one Go
implementation registered against it:

```yaml
# rules/data/R1xx-01.yaml
id: R1xx-01
slug: plural-collection
category: resources
severity: WARN
summary: Collection paths use plural nouns — /book reads as one item, /books as the collection (AIP-131, AIP-122).
docPath: docs/rules/R1xx-01-plural-collection.md
exampleGood: |
  @GetMapping("/books/{id}")
  public BookDto get(@PathVariable Long id)
exampleBad: |
  @GetMapping("/book/{id}")
  public BookDto get(@PathVariable Long id)
```

The catalog and its documentation are mechanically kept in sync:
`internal/cli/rulesdocs_test.go` fails `make ci` when a rule lacks its page in
`docs/rules/` or when an orphan page exists. `vanguard scan --list-rules`
prints the full catalog from the same metadata.

### 1.1 Families

| Family | Domain | Rules |
|---|---|---|
| `R1xx` | Resources & naming | 5 |
| `R2xx` | Methods & HTTP semantics | 6 |
| `R3xx` | Pagination & collections | 3 |
| `R4xx` | Payload & DTO schema | 3 |
| `R5xx` | Errors & status semantics | 3 |
| `R6xx` | Versioning & multi-protocol | 2 + 5 demo fixtures |

Ids are neutral (`R<F>xx-NN`) by design — see the
[charter §3.0](/charter) conventions and the per-family pages under
[Rules](/rules/README).

## 2. Severity Model and the Exit Code

| Severity | Meaning | Exit-code effect |
|---|---|---|
| `ERROR` | Design mistake that breaks the API contract | any surviving finding → **exit 1** |
| `WARN` | Convention violation with evidence | display only — exit 0 |
| `INFO` | Soft recommendation | display only — exit 0 |

| Code | Condition |
|---|---|
| `0` | Scan completed, no surviving ERROR finding (clean, WARN/INFO only, or no API surface) |
| `1` | ≥ 1 surviving ERROR finding (after config + suppression) |
| `2` | Tool error — config unreadable/bad schema, unknown rule option, bad usage; nothing is reported as findings |

Downgrading a rule in config changes whether CI fails; suppressing or
disabling removes the finding from the report entirely. The `--severity`
flag is a display threshold and never changes the exit code. The full
contract is specified in [Configuration §5](/config).

## 3. The Suppression Pipeline

Suppression runs after evaluation, before reporting, in a fixed order:

1. **Config path-scoped suppressions** — `{rule, paths, reason}` entries in
   `.vanguard.yaml`; the rule ref may be an exact id or a family prefix.
2. **Inline directives** — `// vanguard:ignore <rule-id> [reason]` (covers
   the line and the line below), `ignore-begin`/`ignore-end` blocks (nesting
   per rule reference). Inline wins when both layers cover a finding.
3. **Accounting** — suppressed findings are counted (`Report.Suppressed`)
   and noted with rule, source (`inline|config`), reason, and location
   (`Report.SuppressedNotes`). Every output format surfaces the count.

Safety properties (charter §6.4.1):

- a malformed directive suppresses **nothing** and produces a diagnostic;
- diagnostics never abort the scan and never change the exit code;
- an unclosed `ignore-begin` suppresses to end-of-file *and* produces a
  diagnostic — silent over-suppression is impossible to miss at `--verbose`.

## 4. Determinism and Performance

Determinism is an engine contract, not a hope:

- **Byte-identical reports** — same tree, same bytes, every format. Verified
  6/6 sha256-identical pairs on a 10k-file / 353k-LOC Spring tree (W4
  battery, `testdata/perf-results.json`).
- **No wall-clock in output** — the engine-measured duration is excluded
  unless `--timing` is set (test-strategy §5.4).
- **Stable alert identity** — SARIF carries `partialFingerprints` derived
  from (rule, file, line, column, message), so GitHub tracks the same alert
  across pushes instead of reopening it.

Measured performance on the 10k-file / 353k-LOC tree, 2 vCPU:

| Metric | Value |
|---|---|
| p95 scan duration | 39.5 s |
| Cold start | ≤ 50 ms |
| Determinism pairs | 6/6 byte-identical |

## References

- [Rule Catalog](/rules/README)
- [Configuration](/config)
- [Tuning & Suppression](/guides/tuning)
- [Test Strategy](/test-strategy)
