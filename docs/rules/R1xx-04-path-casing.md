# R1xx-04 — path-casing

| | |
|---|---|
| ID | `R1xx-04` |
| Slug | `path-casing` |
| Category | `resources` |
| Default severity | **WARN** |
| AIP reference | AIP-122 |
| Family | R1xx — resources & naming |

## Why

One casing convention across the app: literal path segments are kebab-case
and path variables are lowerCamelCase. Mid-kebab capitals split the family
in two — `/search/all-by-Name` next to seven `/search/all-by-condition`
routes (w1-03 pain 2).

## What fires

- Literal segments that are not `lower-kebab` (`all-by-Name`,
  `loanOrders`, `order_items`) — the suggestion is the kebab-cased path.
- Path variables that are not lowerCamel (`{YouthId}`, `{user_id}`) — the
  suggestion re-spells the variable inside the real path.

## What stays silent (FP shield)

- Lower-case acronyms in kebab (`/api-keys`) and camel variables
  (`{bookId}`).
- Wildcard segments (`*`, `**`, `{*path}`).
- Lower-case dot suffixes (`/reports/export.csv`).

## Spring example — compliant

```java
// kebab segments, camel variable
@GetMapping("/audio-books/{bookId}")
public AudioBookDto get(@PathVariable Long bookId) { ... }
```

## Spring example — violation

```java
// a capital inside the kebab shape, a variable not lowerCamel
@GetMapping("/audioBooks/{BookId}")
public AudioBookDto get(@PathVariable Long bookId) { ... }
```

## Suppression

```java
// vanguard:ignore R1xx-04 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) — useful for a legacy route block you alias rather than
rename.

## Scope notes

- Literal and variable segments are two branches of one rule: a path can
  fire once per offending segment.
- The variable suggestion re-spells the identifier (`{user_id}` →
  `{userId}`); renaming the Java parameter is the client-visible change.
