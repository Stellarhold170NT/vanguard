# R6xx-92 — demo-no-response-type

| | |
|---|---|
| ID | `R6xx-92` |
| Slug | `demo-no-response-type` |
| Category | `demo` |
| Default severity | **WARN** |
| Grounding | charter §3.0 (data-driven proof), w2-07 |
| Family | R6xx demo set — golden-fixture rules |

## Why

The WARN member of the demo set (w2-07): stub and Spring methods without a
declared response type produce real WARN findings, so severity grouping
(ERROR → WARN → INFO, charter §6.6) is observable in golden snapshots.
The defect is real too: clients cannot deserialize an undeclared result.

## What fires

A method whose `Response.Type.Name` is empty — a void endpoint, an
endpoint whose return type the adapter could not resolve into a name, or
a stub operation without a `response` block.

## What stays silent

Every method that declares a response type (`ResponseEntity<Void>`
unwraps to an empty body type and DOES fire — declare a real return type
or a typed wrapper instead).

## Spring example — compliant

```java
// the result is declared — clients can deserialize it
@DeleteMapping("/members/{memberId}")
public MemberDto deleteMember(@PathVariable Long memberId) { ... }
```

## Spring example — violation

```java
// no response type — what does the client get back?
@PostMapping("/orders/{orderId}/archive")
public void archiveOrder(@PathVariable Long orderId) { ... }
```

## Suppression

```java
// vanguard:ignore R6xx-92 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for endpoints where "no body" is the contract and the
void is deliberate.

## Scope notes

- The rule reads the IR only; a void method IS an undeclared response at
  the wire level — the message names the operation so the reader can
  decide.
- Same fixture family as R6xx-91/93/94; the golden showcase
  (`testdata/golden/java-spring/`) fires it twice, the clean case never.
