# R6xx-91 — demo-bad-method-name

| | |
|---|---|
| ID | `R6xx-91` |
| Slug | `demo-bad-method-name` |
| Category | `demo` |
| Default severity | **ERROR** |
| Grounding | charter §3.0 (data-driven proof), w2-07 |
| Family | R6xx demo set — golden-fixture rules |

## Why

One of the five demo rules that made the W2 pipeline observable end to
end (w2-07): a real ERROR finding from a real adapter surface, so the
§6.6 severity grouping and the exit-1 plumbing are pinned in golden
snapshots rather than mocked. The naming defect it encodes is real
anyway: one canonical operation name per endpoint.

## What fires

The method's operation name carries the `demoBad` fixture marker
(`demoBadGetOrder`) — the marker the w2-07 fixtures plant. The suggestion
strips the marker and lower-cases what remains (`getOrder`): replacement
text, not advice.

## What stays silent

Any method whose name does not start with `demoBad`. The check reads only
`ir.Method.OperationName` — no source, no reflection.

## Spring example — compliant

```java
// the operation is named after what it does
@GetMapping("/orders/{orderId}")
public OrderDto getOrder(@PathVariable Long orderId) { ... }
```

## Spring example — violation

```java
// the fixture marker leaks into the operation name
@GetMapping("/orders/{orderId}/")
public void demoBadGetOrder(@PathVariable Long orderId) { ... }
```

## Suppression

```java
// vanguard:ignore R6xx-91 <reason>
```

The demo family exists to be exercised by the golden corpus
(`testdata/golden/java-spring/`); suppressing it in application configs is
the normal case (`R6xx` family disable) if you scan production repos.

## Scope notes

- Demo rules are first-class registry members: same §5.4 validation, same
  catalog (`--list-rules`), same suppression paths — nothing is
  special-cased.
- See `testdata/golden/java-spring/README.md` for the rule → fixture map.
