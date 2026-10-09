# R6xx-99 — demo-trailing-slash

| | |
|---|---|
| ID | `R6xx-99` |
| Slug | `demo-trailing-slash` |
| Category | `demo` |
| Default severity | **ERROR** |
| Grounding | charter §3.0 (data-driven proof), w2-04 |
| Family | R6xx demo set — golden-fixture rules |

## Why

The w2-04 data-driven acceptance fixture: the rule's identity lives in a
YAML record joined to a Go check by id — "metadata là data" proven at the
smallest possible scale. The defect is canonical-form hygiene: a path and
its trailing-slash twin are two URLs for one resource.

## What fires

A method whose full IR path ends with `/` (`/api/v1/orders/{orderId}/`).
The suggestion is the same path with the slash trimmed.

## What stays silent

- Every normalized path. **Both adapters normalize paths by contract** —
  the spring adapter's `mergePath` joins with `path.Join` and the stub
  merge cleans trailing slashes (IR contract) — so NO fixture repo can
  express this trigger. The rule is pinned by `rules/demo_test.go` on
  hand-built IR, and the w3-07 coverage test carries it as the one
  documented exemption (`internal/golden/rule_coverage_test.go`).

## Spring example — compliant

```java
// one canonical form
@GetMapping("/orders/{orderId}")
public OrderDto getOrder(@PathVariable Long orderId) { ... }
```

## Spring example — violation

```java
// hand-built IR shape (the adapter normalizes this away):
//   Path: "/api/v1/orders/{orderId}/"
//
// a raw mapping like the below never reaches the IR with its slash
@GetMapping("/orders/{orderId}/")
public void demoBadGetOrder(@PathVariable Long orderId) { ... }
```

## Suppression

```java
// vanguard:ignore R6xx-99 <reason>
```

Because the trigger cannot survive the adapters in v0.1, suppression is a
unit-level concern only; application configs typically disable the demo
family (`R6xx`) instead.

## Scope notes

- ERROR severity: like R6xx-91, this rule exists so the exit-1 plumbing
  has a stub-reachable trigger in the W2 fixtures.
- If a future adapter preserves raw paths, move the coverage from the
  unit test into the golden corpus and drop the exemption in
  `rule_coverage_test.go`.
