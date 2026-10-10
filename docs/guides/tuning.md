---
title: Tuning & Suppression
description: Keep Vanguard's signal high on a real codebase — severity overrides, path-scoped suppressions, inline ignores, and the demo-fixture rules to disable.
---

# Tuning & Suppression

Gate fatigue kills linters. Vanguard's answer is not "fewer rules" but
**tunable, accountable noise control**: every downgrade, disable, and
suppression is recorded with a reason and stays visible in the report. This
guide is the practical walkthrough; the normative schema is
[Configuration](/config).

## 1. Disable the Demo-Fixture Rules Outside Testdata

The `R6xx-9x` family (91, 92, 93, 94, 99) exists to exercise the golden
fixtures and the reporting pipeline. On a real service they fire on
perfectly ordinary code — on the real-fire baseline they produced 14 of the
findings. Disable them for everyday scans:

```yaml
version: 1
rules:
  R6xx-91: { disabled: true }
  R6xx-92: { disabled: true }
  R6xx-93: { disabled: true }
  R6xx-94: { disabled: true }
  R6xx-99: { disabled: true }
```

Or the whole family at once (`R6xx: { disabled: true }` disables
[R6xx-01](/rules/R6xx-01-versioned-path) and
[R6xx-02](/rules/R6xx-02-grpc-standard-methods) too — re-enable those
individually; the exact id wins over the family prefix).

## 2. Downgrade Before You Disable

A rule that is directionally right but too loud for your codebase usually
needs a severity change, not deletion. Downgrading to `WARN`/`INFO` keeps the
finding visible while removing its ability to fail CI:

```yaml
rules:
  R1xx-02: { severity: WARN }   # was ERROR — still reported, no longer gates
  R2xx-05: { severity: INFO }   # verification hints stay visible, quieter
```

The exit code only reacts to surviving **ERROR** findings
([contract](/config)).

## 3. Suppress by Path, With a Reason

Path-scoped suppressions are for code you cannot fix soon but can name:

```yaml
suppressions:
  - rule: R3xx-02
    paths: ["**/generated/**"]
    reason: "generated DTOs — refactor planned for next quarter"
  - rule: R1xx
    paths: ["src/legacy/**"]
    reason: "legacy module, frozen until migration W3"
```

- `rule` is an exact id or family prefix; `paths` are globs relative to the
  config file.
- Suppressed findings disappear from the findings list but **not from the
  report**: every format prints the suppressed count, and `--verbose` lists
  each note with its reason. A silent suppression wave is therefore visible
  in the gate output (`N suppressed`).

## 4. Inline Ignores for Point Fixes

For a single false positive at a single location, keep the decision next to
the code:

```java
// vanguard:ignore R1xx-01 tree is an action segment, not a collection noun
@GetMapping("/department/tree")
public List<DepartmentTreeResponse> tree() { ... }

// vanguard:ignore-begin R3xx-02 report export, binary payload — envelope N/A
@GetMapping("/reports/export")
public byte[] exportByCondition() { ... }
// vanguard:ignore-end R3xx-02
```

- A line directive covers its own line **and the line below** (the directive
  normally sits above the declaration).
- The rule reference may be a family prefix; blocks nest per reference.
- **Fail-open**: a malformed directive suppresses nothing and produces a
  diagnostic — a typo can never quietly hide a finding.

## 5. A Worked Tuning Pass

The first scan of a real Spring service typically clusters noise in a few
identifiable places (measured on the
[real-fire baseline](/guides/case-study)):

| Noise source | Symptom | First move |
|---|---|---|
| Demo rules `R6xx-9x` | findings on ordinary endpoints | disable the family |
| Action-shaped paths (`/tree`, `/export`, `/all-by-condition`) | `R1xx-01` singular-collection, `R2xx-05` action-on-GET | inline ignore per span, or path suppression for the legacy module |
| Binary exports (`byte[]`) | `R3xx-02` bare-array envelope | inline ignore with reason |
| Generated DTOs | `R4xx-02` casing | path suppression `**/generated/**` |

The engine-side fixes for these heuristics are tracked in the backlog; until
they land, config tuning is the supported path — and the suppression counts
keep the tune-down visible to the team.

## 6. What Not To Do

- **Don't disable a whole family to silence one rule.** Use the exact id;
  family prefixes are for deliberate, repo-wide decisions.
- **Don't suppress without a reason.** The reason is what makes the next
  reviewer's audit possible.
- **Don't treat exit 2 as a violation.** A red job from a broken
  `.vanguard.yaml` is a config fix, not a design regression
  ([CI Integration §3](/ci-integration)).
