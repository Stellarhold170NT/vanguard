# R3xx-01 — list-paginated (WARN)

A GET endpoint that returns a collection must accept pagination parameters
(AIP-158). An unpaginated collection is one traffic spike away from
serializing the whole table (w1-03 pain 4).

## Fires when

ALL of these hold on the same `ir.Method`:

1. `Verb == GET` (gRPC and non-GET verbs are out of scope);
2. `Response.IsCollection` is true and the response type is non-empty —
   the adapter sets this for `List<T>`, `Page<T>`, arrays, `Flux<T>`…;
3. the path is a **collection path**: the last non-empty path segment
   contains no template variable (`/orders` yes, `/orders/{id}` no — a
   trailing slash does not change the verdict);
4. `Pagination.Style` is neither `pageable` nor `params` — anything the
   adapter did not explicitly declare as paginated counts as unpaginated.

## Suggestion

`add a Pageable parameter or page/size query parameters to bound the
result set`

## Scope notes (attack surface — W4, read this)

- The rule never inspects the response element count or any body — it is a
  signature-level check over the IR.
- `POST /search`-style endpoints returning collections are out of scope
  (verb must be GET). A search endpoint with the same dump risk is the
  charter's R3xx-05/06 backlog, not this rule.
- Export endpoints (`/export`) are not exempted in v0.1; they will fire.
  Suppress per-path with a config suppression if exports are intentional.

## Good / Bad

```java
// good — paginated
@GetMapping
public Page<BookDto> list(Pageable pageable) { ... }

// bad — full table on one request
@GetMapping
public List<BookDto> list() { ... }
```
