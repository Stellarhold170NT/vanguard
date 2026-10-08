# `.vanguard.yaml` — config schema v1 (w2-04)

This document is the detailed specification of the config file that
`internal/engine` reads (charter §6.4). A commented example lives at
[`.vanguard.example.yaml`](../.vanguard.example.yaml) in the repository root;
`vanguard init` (w2-06) will generate the same shape with the full rule-id
list.

## 1. Discovery

- The file is named `.vanguard.yaml`.
- The engine looks for it starting at the scan root and **walks up the
  directory tree to the filesystem root** (golangci-lint / protolint style,
  charter §6.4.2 #1). The **nearest** file wins.
- **No multi-level merging**: one file is in effect per scan — the nearest
  one. A config higher up the tree is ignored entirely when a nearer one
  exists.
- Not finding any config is not an error: the scan runs with defaults (all
  non-default-disabled rules, `include: ["**"]`, no exclude, no
  suppressions).
- The CLI layer (w2-06) adds `--config FILE` (overrides discovery) and
  `--no-config` (disables config entirely), and `--verbose` prints the
  config path in use.

## 2. Schema (version 1)

```yaml
version: 1                          # REQUIRED; only 1 is accepted

frameworks:                         # OPTIONAL; empty/absent = auto-detect
  java: [spring-boot-mvc]           # language -> adapter ids (wave-2 adds more)

include: ["**"]                     # OPTIONAL; default ["**"]
exclude:                            # OPTIONAL; applied after the discovery
  - "**/generated/**"               # built-in ignores
  - "vendor/**"

rules:                              # OPTIONAL per-rule overrides, keyed by
  R1xx:                             #   family prefix (R1xx) or
    severity: WARN                  #   exact id (R1xx-02)
  R6xx-01:                          # rule default-OFF (charter §3.6)
    disabled: false
    options:                        # only keys declared in the rule metadata
      versionPattern: "/v[0-9]+"

suppressions:                       # OPTIONAL path-scoped suppressions
  - rule: R1xx                      # id or family prefix (REQUIRED)
    paths: ["src/legacy/**"]        # globs (REQUIRED, ≥1 entry)
    reason: "legacy module"         # optional, shown by --verbose
```

### 2.1 Keys

| Key            | Type                          | Default            | Notes |
|----------------|-------------------------------|--------------------|-------|
| `version`      | integer                       | — (required)       | Only `1` is accepted in v0.1. |
| `frameworks`   | map `lang: [adapter,...]`     | `{}` = auto-detect | Consumed by the adapter registry (w2-03); the engine only parses and carries it. |
| `include`      | list of globs                 | `["**"]`           | Files (relative to the config directory) that may be scanned. |
| `exclude`      | list of globs                 | `[]`               | Applied after the discovery built-in ignores (w2-03). |
| `rules`        | map `rule-ref → overrides`    | `{}`               | See §3. Unknown sub-keys are rejected. |
| `suppressions` | list of `{rule, paths, reason}` | `[]`             | See §4. |

No other top-level keys exist; an unknown key is a schema error (§6).

### 2.2 Glob dialect

All globs — `include`, `exclude` and suppression `paths` — are **relative to
the directory that contains the config file** (oasdiff lesson, w1-02 #6) and
use `/` separators on every platform:

- `**` spans **whole path segments only** (`src/**`, `**/generated/**`, `**`).
- `*`, `?` and `[...]` stay within one segment (`path.Match` semantics):
  `*.java` matches `a.java` but not `dir/a.java`; `**/*.java` matches both.
- A malformed pattern (e.g. an unterminated `[ab`) is rejected **at config
  load time** — a typo must fail the config (exit 2), never silently match
  nothing at scan time.

## 3. Rule overrides

Each key under `rules` is either an exact rule id (`R1xx-02`) or a family
prefix (`R1xx`); keys outside the `R1xx..R6xx` taxonomy (charter §3.0) are a
schema error.

| Sub-key    | Type                  | Meaning |
|------------|-----------------------|---------|
| `severity` | `ERROR` \| `WARN` \| `INFO` (any case, normalized) | Replaces the rule's metadata default severity. |
| `disabled` | boolean               | Tri-state: absent = keep the metadata default; `true`/`false` = explicit. |
| `options`  | map `name → value`    | Only keys the rule metadata declares; an unknown option key is a **tool error** (exit 2) reported by the engine at scan time. |

### 3.1 Merge precedence (charter §6.4.2 #4)

For one rule, the effective override is resolved as:

1. the **family prefix** entry applies first (`R1xx`),
2. the **exact id** entry applies second (`R1xx-02`) and wins on every
   conflict,
3. metadata defaults fill whatever config left unspecified.

This holds for `severity`, `disabled` and `options` (options merge key by
key, exact wins per key). Disable matching runs before enable matching, so
`rules: {R1xx: {disabled: true}, R1xx-02: {disabled: false}}` leaves
R1xx-02 **enabled** while every other R1xx rule is disabled — and the
mirror image disables exactly one rule of an enabled family.

A disabled rule is a *config decision*: it produces neither findings nor
suppression notes (contrast §4).

## 4. Suppressions

### 4.1 Path-scoped (config)

Each `suppressions` entry suppresses a finding when **both** hold:

- the entry's `rule` matches the finding's rule id — exact id or family
  prefix, and
- one of the entry's `paths` globs matches the finding's file (relative to
  the config directory, dialect §2.2).

`reason` is optional and is surfaced by `--verbose` (w2-06) as the note's
reason. Invalid rule references and malformed globs are rejected at load
time (exit 2).

### 4.2 Inline (source comments)

```
// vanguard:ignore <rule-id> [reason]          — covers this line and the line below
// vanguard:ignore-begin <rule-id> [reason]    — opens a block
// vanguard:ignore-end <rule-id>               — closes the innermost open block for <rule-id>
```

- `<rule-id>` may be an exact id or a family prefix; a family prefix covers
  every rule of the family.
- A line directive covers findings on its own line **and on the line
  immediately below** (the directive typically sits above the declaration).
- Blocks cover every line between `begin` and `end`, inclusive. Blocks nest
  per rule reference; an `end` closes the most recent open block for the
  same reference.
- Matching is per file — directives never leak across files.
- **Fail-open**: a malformed directive (bad rule reference, missing space)
  suppresses **nothing** and produces a diagnostic anchored at the comment.
  An `ignore-begin` without a matching `ignore-end` keeps suppressing to
  end-of-file (lenient) **and** produces a diagnostic; a stray
  `ignore-end` produces a diagnostic and suppresses nothing. Diagnostics
  never abort the scan and never change the exit code (charter §5.3).

### 4.3 Precedence between the layers

Inline wins: when an inline directive and a config path-suppression both
cover a finding, the verbose note credits `inline`. A suppressed finding is
counted (`Report.Suppressed`) and noted (`Report.SuppressedNotes`: rule,
source `inline|config`, reason, location) instead of being carried.

## 5. Severity and the exit code

| Severity | Meaning (charter §3.0)                    | Exit-code effect |
|----------|-------------------------------------------|------------------|
| `ERROR`  | design mistake that breaks the API contract | any surviving finding → **exit 1** |
| `WARN`   | convention violation with evidence         | display only — **exit 0** |
| `INFO`   | soft recommendation                        | display only — **exit 0** |

Full CI contract (charter §6.3, `engine.ExitCode`, matched 1:1 by the w2-06
CLI):

| Code | Condition |
|------|-----------|
| `0`  | scan completed, no surviving `ERROR` finding (clean, only WARN/INFO, or no API surface found) |
| `1`  | ≥ 1 surviving finding with severity `ERROR` (after config + suppression) |
| `2`  | tool error: config unreadable, config bad schema, unknown rule option, internal error — nothing is reported as findings |

Downgrading a rule's severity in config (`severity: WARN`) therefore changes
whether the scan fails CI; suppressing or disabling does not print findings
at all. The `--severity` display flag (w2-06) is a display threshold and
never changes the exit code.

## 6. Error contract

Every config parse/validation failure produces an error of the form

```
.vanguard.yaml:<line>: <what is wrong>
```

— the config file's **base name**, the **1-based line of the offending
key**, then the reason. Examples:

```
.vanguard.yaml:1: missing required key "version"
.vanguard.yaml:2: unknown key "rulez" (known: version, frameworks, include, exclude, rules, suppressions)
.vanguard.yaml:4: invalid severity "FATAL" (want ERROR, WARN or INFO)
.vanguard.yaml:3: duplicate key "version"
```

YAML syntax errors surface under the same `file:line` contract. All of
these are the exit-2 tool-error class: the CLI prints them to stderr and
never prints findings (charter §6.3).
