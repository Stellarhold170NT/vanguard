# R3xx-03 — unbounded-page-size (INFO)

A size/limit query parameter without an upper bound lets one client
request unbounded pages — a mild DoS surface (AIP-158, w1-03 pain 4:
Pageable is used widely, but there is no evidence of a cap).

## Fires when

ALL of these hold on the same `ir.Param`:

1. `In == query` (the finding anchors on the parameter itself);
2. the parameter name is **size-carrying**: the lower-cased name, or the
   last camelCase word of it, is `size` or `limit` (`size`, `limit`,
   `pageSize`, `maxSize` qualify; `page`, `offset` do not);
3. `Validation` — the raw constraint annotations the adapter collected —
   contains no `@Max` (any `@Max(...)` spelling counts; `@Min`/`@Size`
   alone do not cap a page size).

## Stays silent when

- the parameter carries `@Max` (e.g. `@Max(100)`);
- the method pages through a framework `Pageable` argument instead of an
  explicit size parameter. **This is a documented IR limit, not an
  endorsement**: `@PageableDefault` sets a default, not a cap (w3-02
  adapter limitation), and a `Pageable` has no `ir.Param` of its own to
  anchor on. Caps for framework Pageables live in a custom resolver the
  IR cannot see.

## Scope notes (attack surface — W4, read this)

- The cap signal is the literal substring `@Max` in `Param.Validation` —
  the adapter joins every non-binding annotation's raw text there. A
  custom `@PageLimit` annotation is invisible to v0.1.
- Header/path parameters named `size` are out of scope (query only).
- Word-boundary matching comes from camelCase splitting, so `sizer` or
  `household` do not qualify.

## Good / Bad

```java
// good — capped
@GetMapping("/search")
public List<BookDto> search(@RequestParam("size") @Max(100) int size) { ... }

// bad — unbounded
@GetMapping("/search")
public List<BookDto> search(@RequestParam("size") int size) { ... }
```
