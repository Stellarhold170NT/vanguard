# R3xx-03 — unbounded-page-size

| | |
|---|---|
| ID | `R3xx-03` |
| Slug | `unbounded-page-size` |
| Category | `pagination` |
| Default severity | **INFO** |
| AIP reference | AIP-158 |
| Family | R3xx — pagination |

## Why

A size/limit query parameter without an upper bound lets one client
request unbounded pages — a mild DoS surface (AIP-158, w1-03 pain 4:
Pageable is used widely, but there is no evidence of a cap).

## What fires

ALL of these hold on the same `ir.Param`:

1. `In == query` (the finding anchors on the parameter itself);
2. the parameter name is **size-carrying**: the lower-cased name, or the
   last camelCase word of it, is `size` or `limit` (`size`, `limit`,
   `pageSize`, `maxSize` qualify; `page`, `offset` do not);
3. `Validation` — the raw constraint annotations the adapter collected —
   contains no `@Max` (any `@Max(...)` spelling counts; `@Min`/`@Size`
   alone do not cap a page size).

## What stays silent (FP shield)

- The parameter carries `@Max` (e.g. `@Max(100)`).
- The method pages through a framework `Pageable` argument instead of an
  explicit size parameter — **a documented IR limit, not an endorsement**:
  `@PageableDefault` sets a default, not a cap, and a `Pageable` has no
  `ir.Param` of its own to anchor on.
- Header/path parameters named `size` (query only); word-boundary
  splitting keeps `sizer` or `household` out.

## Spring example — compliant

```java
// The cap is declared next to the parameter.
@GetMapping("/search")
public List<BookDto> search(@RequestParam("size") @Max(100) int size) { ... }
```

## Spring example — violation

```java
// No upper bound — one client can request the whole table.
@GetMapping("/books")
public List<BookDto> listAllBooks(@RequestParam int page, @RequestParam int size) { ... }
```

## Suppression

```java
// vanguard:ignore R3xx-03 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1).

## Scope notes

- The cap signal is the literal substring `@Max` in `Param.Validation` —
  a custom `@PageLimit` annotation is invisible to v0.1.
- Caps for framework Pageables live in a custom resolver the IR cannot
  see — a known limit, W4 attack surface.
