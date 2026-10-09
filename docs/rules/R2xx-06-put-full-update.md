# R2xx-06 — put-full-update

| | |
|---|---|
| ID | `R2xx-06` |
| Slug | `put-full-update` |
| Category | `methods` |
| Default severity | **WARN** |
| AIP reference | AIP-134 |
| Family | R2xx — methods & verbs |

## Why

PUT means full replacement (AIP-134). When the PUT body is typed as a
DIFFERENT type from the response — `CogUpdateRequest` in, `CogDto` out —
the request carries a subset and the verb overstates it: a partial update
done through the wrong verb (w1-03 pain 7c, `partialUpdateYouth`).

## What fires

ALL of these hold on the same `ir.Method`:

1. `Verb == PUT` (HTTP; rpc never matches);
2. the method declares a payload (`Payload != nil`);
3. the declared response type is non-empty AND `Payload.Name` differs
   from `Response.Type.Name` case-insensitively.

Matching types (full replacement) or an undeclared response stay silent.

## What stays silent (FP shield)

- `PUT` with the same type in and out — the compliant full-replacement
  shape (the w3-07 clean fixture pins it).
- Undeclared response types; PUTs without a payload.
- Transport wrappers on either side are unwrapped by the w3-02 adapter.

## Spring example — compliant

```java
// The same type in and out — full replacement.
@PutMapping("/books/{bookId}")
public BookDto updateBook(@RequestBody BookDto dto, @PathVariable Long bookId) { ... }
```

## Spring example — violation

```java
// A dedicated request type on PUT — partial-update semantics.
@PutMapping("/members/{memberId}")
public MemberDto updateMember(@RequestBody MemberUpdateRequest request, @PathVariable Long memberId) { ... }
```

## Suppression

```java
// vanguard:ignore R2xx-06 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for a command-style request DTO that still replaces
the whole resource.

## Scope notes

- Symmetric to R2xx-03 patch-partial (same type in/out on PATCH vs
  different type on PUT) — the pair covers the verb/body agreement in
  both directions.
- A request DTO that deliberately differs while carrying every field is
  indistinguishable at the signature level; the message names both types.
