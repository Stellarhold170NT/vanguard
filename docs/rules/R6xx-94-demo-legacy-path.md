# R6xx-94 — demo-legacy-path

| | |
|---|---|
| ID | `R6xx-94` |
| Slug | `demo-legacy-path` |
| Category | `demo` |
| Default severity | **INFO** |
| Grounding | charter §3.0 (data-driven proof), w2-07 |
| Family | R6xx demo set — golden-fixture rules |

## Why

The w2-07 data-driven acceptance artifact: this rule shipped with ONE
YAML record and ONE registry row — no engine, renderer, or harness change
— proving "a rule is data" (charter §3.0). The defect it encodes is
transitional debt: `/legacy/` paths are migration leftovers that should
move to their versioned route.

## What fires

A method whose full path contains the segment `/legacy/` — the marker the
w2-07 fixtures plant. The suggestion moves the route under its versioned
prefix.

## What stays silent

Every path without a `/legacy/` segment. Substring containment is judged
on the merged IR path (base + method mapping).

## Spring example — compliant

```java
// the route carries its version
@RestController
@RequestMapping("/api/v1/reports")
public class ReportController { ... }
```

## Spring example — violation

```java
// a transitional /legacy/ path still serving traffic
@RestController
@RequestMapping("/legacy/reports")
public class LegacyReportController { ... }
```

## Suppression

```java
// vanguard:ignore R6xx-94 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for a legacy module with a planned refactor window.

## Scope notes

- The check is `strings.Contains(path, "/legacy/")` — a path segment, not
  a route-style judgement; `/my-legacy-reports` contains the substring
  and fires (documented sharp edge, demo severity).
- Related production rule: R6xx-01 versioned-path (opt-in) flags the
  missing version segment itself; in the w3-07 showcase both fire on the
  same service — one from the path marker, one from the policy.
