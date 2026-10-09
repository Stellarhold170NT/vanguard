# R1xx-02 — no-verb-path

| | |
|---|---|
| ID | `R1xx-02` |
| Slug | `no-verb-path` |
| Category | resources |
| Default severity | **ERROR** |
| Inspired by | AIP-131, AIP-133, AIP-135 |
| Docs stub | w3-03 — full page ships with w3-07 |

## Why

The HTTP verb already carries the action. A CRUD verb inside the path
duplicates it and forks the surface: the same delete exists twice —
`DELETE /records/{id}` next to `@DeleteMapping("/delete/{id}")` (w1-03 pain
1+3: 5× `@DeleteMapping("/delete")`, `/update/draft`, `/update/submit`).

## Good / Bad

```java
// Bad — verb stated twice
@DeleteMapping("/delete/{id}")
public void remove(@PathVariable Long id)

// Good — the HTTP verb is the action
@DeleteMapping("/records/{id}")
public void remove(@PathVariable Long id)
```

## What fires (exact tokens only)

- Kebab or camel tokens matching a CRUD verb: `delete`, `update`, `get`,
  `create`, `save`, … — including inside one segment (`/getUserById` →
  `get`).
- The suggestion names the HTTP shape that replaces it (e.g. `PUT
  /<collection>/{id}` for `update`).

## What stays silent (FP shield)

- Substrings and derived nouns: `getter`, `updates`, `preview`,
  `rebuild` — token-boundary matching only.
- Non-CRUD action verbs (`search`, `export`, `generate`, `checkout`) — they
  are R2xx-05 `custom-method-post`'s subject.

## Suppress

```java
// vanguard:ignore R1xx-02 <reason>
```
