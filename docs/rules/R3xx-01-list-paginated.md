# R3xx-01 — list-paginated

| | |
|---|---|
| ID | `R3xx-01` |
| Slug | `list-paginated` |
| Category | `pagination` |
| Default severity | **WARN** |
| AIP reference | AIP-158 |
| Family | R3xx — pagination |

## Why

A GET endpoint that returns a collection must accept pagination parameters
(AIP-158). An unpaginated collection is one traffic spike away from
serializing the whole table (w1-03 pain 4).

## What fires

ALL of these hold on the same `ir.Method`:

1. `Verb == GET` (gRPC and non-GET verbs are out of scope);
2. `Response.IsCollection` is true with a non-empty response type —
   `List<T>`, `Page<T>`, arrays, `Flux<T>`…;
3. the path is a **collection path**: the last non-empty segment contains
   no template variable (`/orders` yes, `/orders/{id}` no);
4. `Pagination.Style` is neither `pageable` nor `params` — anything the
   adapter did not explicitly declare as paginated counts as unpaginated.

## What stays silent (FP shield)

- A Spring `Pageable` argument → style `pageable`.
- Explicit `page`/`size` query parameters → style `params`.
- Item routes (`/orders/{id}`) and non-GET collection returns.

## Spring example — compliant

```java
// The Pageable parameter bounds the result set.
@GetMapping("/members")
public Page<MemberDto> listMembers(Pageable pageable) { ... }
```

## Spring example — violation

```java
// A collection response with no way to bound it.
@GetMapping("/members/export")
public List<MemberDto> exportMembers() { ... }
```

## Suppression

```java
// vanguard:ignore R3xx-01 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — for endpoints whose collection is bounded by design
(e.g. a fixed dictionary list).

## Scope notes

- Signature-level check: the rule never inspects element counts or
  bodies.
- `POST /search`-style endpoints returning collections are out of scope
  (verb must be GET) — a documented miss, W4 attack surface.
