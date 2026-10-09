# R2xx-03 — patch-partial

| | |
|---|---|
| ID | `R2xx-03` |
| Slug | `patch-partial` |
| Category | `methods` |
| Default severity | **WARN** |
| AIP reference | AIP-134 |
| Family | R2xx — methods & verbs |

## Why

PATCH means partial update. When the PATCH body is typed as the full
entity (the same type the endpoint returns), the verb and the body
disagree — the request reads as full replacement done through the wrong
verb (AIP-134; w1-03 pain 7c, `partialUpdateYouth`).

## What fires

ALL of these hold on the same `ir.Method`:

1. `Verb == PATCH` (HTTP; rpc never matches);
2. the method declares a payload (`Payload != nil`);
3. the declared response type is non-empty AND `Payload.Name` equals
   `Response.Type.Name` case-insensitively — the same type in and out
   reads as full-entity replacement.

## What stays silent (FP shield)

- A PATCH whose payload type differs from the response type (a partial
  DTO) — the heuristic cannot establish "full".
- An undeclared response, or a PATCH without a body.
- Transport wrappers (`ResponseEntity<...>`, `Mono<...>`) are already
  unwrapped by the w3-02 adapter, so the comparison sees body types.

## Spring example — compliant

```java
// A dedicated partial payload — the verb and the body agree.
@PatchMapping("/members/{memberId}")
public MemberDto patchMember(@RequestBody MemberPatch patch, @PathVariable Long memberId) { ... }
```

## Spring example — violation

```java
// The body is the full MemberDto — PUT semantics on a PATCH verb.
@PatchMapping("/members/{memberId}")
public MemberDto patchMember(@RequestBody MemberDto fullEntity, @PathVariable Long memberId) { ... }
```

## Suppression

```java
// vanguard:ignore R2xx-03 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for a PATCH that legitimately accepts the full
representation and applies it partially.

## Scope notes

- Signature-level heuristic: a same-type PATCH that genuinely applies a
  subset of fields is indistinguishable — the message names the type so
  the reader can verify.
- Symmetric rule: R2xx-06 put-full-update fires on the mirror defect
  (PUT with a different-type payload).
