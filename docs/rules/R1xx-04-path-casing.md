# R1xx-04 — path-casing

| | |
|---|---|
| ID | `R1xx-04` |
| Slug | `path-casing` |
| Category | resources |
| Default severity | **WARN** |
| Inspired by | AIP-122 (camel-case-uris, re-expressed for source) |
| Docs stub | w3-03 — full page ships with w3-07 |

## Why

One casing convention across the app: literal path segments are kebab-case
and path variables are lowerCamelCase. Mid-kebab capitals split the family
in two — `/search/all-by-Name` next to seven `/search/all-by-condition`
routes (w1-03 pain 2).

## Good / Bad

```java
// Good — kebab segments, camel variable
@GetMapping("/audio-books/{bookId}")

// Bad — a capital inside the kebab segment
@GetMapping("/audioBooks/{BookId}")
public AudioBookDto get(@PathVariable Long bookId)
```

## What fires

- Literal segments that are not `lower-kebab` (`all-by-Name`, `loanOrders`,
  `order_items`) — the suggestion is the kebab-cased path.
- Path variables that are not lowerCamel (`{YouthId}`, `{user_id}`).

## What stays silent (FP shield)

- Lower-case acronyms in kebab (`/api-keys`) and camel variables
  (`{bookId}`).
- Wildcard segments (`*`, `**`, `{*path}`).
- Lower-case dot suffixes (`/reports/export.csv`).

## Suppress

```java
// vanguard:ignore R1xx-04 <reason>
```
