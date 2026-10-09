# R6xx-93 — demo-get-with-body

| | |
|---|---|
| ID | `R6xx-93` |
| Slug | `demo-get-with-body` |
| Category | `demo` |
| Default severity | **INFO** |
| Grounding | charter §3.0 (data-driven proof), w2-07 |
| Family | R6xx demo set — golden-fixture rules |

## Why

The INFO member of the demo set (w2-07): GET methods carrying a payload
produce real INFO findings, completing the severity taxonomy in golden
snapshots. The defect mirrors the production rule R2xx-01 (ERROR) at demo
severity — same transport fact, one grading for the fixture corpus and
one for real repos.

## What fires

A method with `Verb == GET` and a non-nil `Payload` — the same
signature-level reading as R2xx-01's payload half.

## What stays silent

GETs with query parameters only, every non-GET verb, gRPC operations.

## Spring example — compliant

```java
// the query travels in the URL
@GetMapping("/members/auto-complete")
public MemberSuggestions autoComplete(@RequestParam String term) { ... }
```

## Spring example — violation

```java
// a body on GET — many clients silently drop it
@GetMapping("/members/auto-complete")
public MemberSuggestions autoComplete(@RequestBody MemberQuery query) { ... }
```

## Suppression

```java
// vanguard:ignore R6xx-93 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1). In production repos, disable the demo family
(`R6xx-9x`) and let R2xx-01 grade the defect at ERROR instead.

## Scope notes

- The demo set is registered next to the real families on purpose: the
  w2-07 golden corpus pinned all three severities before any real rule
  existed, and the w3-07 showcase re-proves it per rule.
- A GET with a body fires BOTH R2xx-01 (ERROR) and this rule (INFO) —
  the demo copy is fixture material, not a double-report bug.
