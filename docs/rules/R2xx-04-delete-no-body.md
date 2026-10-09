# R2xx-04 — delete-no-body

| | |
|---|---|
| ID | `R2xx-04` |
| Slug | `delete-no-body` |
| Category | `methods` |
| Default severity | **WARN** |
| AIP reference | AIP-135 |
| Family | R2xx — methods & verbs |

## Why

A DELETE body is legal HTTP but unreliable in practice: popular clients
and intermediaries drop or reject it, so bulk deletes travel through a
channel many clients cannot send (w1-03 pain 7b: `@DeleteMapping` +
`@RequestBody List<String> ids`).

## What fires

`Verb == DELETE` and the method declares a request body — either view of
the w2-02 invariant (`Payload` non-nil or an `in=body` Param), the same
mechanism as R2xx-01.

## What stays silent (FP shield)

- DELETE with query parameters (`?ids=1,2`) or a batch path
  (`POST /books:batchDelete`) — the compliant bulk shapes.
- gRPC operations never match (HTTP verb semantics).

## Spring example — compliant

```java
// The identifiers travel where every client can send them.
@DeleteMapping("/members/{memberId}")
public MemberDto deleteMember(@PathVariable Long memberId) { ... }
```

## Spring example — violation

```java
// A body on DELETE — dropped by popular HTTP clients.
@DeleteMapping("/members/{memberId}")
public MemberDto deleteMember(@RequestBody List<Long> memberIds) { ... }
```

## Suppression

```java
// vanguard:ignore R2xx-04 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for stacks that genuinely require DELETE bodies.

## Scope notes

- One severity softer than R2xx-01 (WARN vs ERROR): a DELETE body is
  legal HTTP; the convention just routes bulk deletes elsewhere.
- None known for v0.1 (same measured surface as R2xx-01).
