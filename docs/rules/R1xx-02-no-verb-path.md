# R1xx-02 — no-verb-path

| | |
|---|---|
| ID | `R1xx-02` |
| Slug | `no-verb-path` |
| Category | `resources` |
| Default severity | **ERROR** |
| AIP reference | AIP-131, AIP-133, AIP-135 |
| Family | R1xx — resources & naming |

## Why

The HTTP verb already carries the action. A CRUD verb inside the path
duplicates it and forks the surface: the same delete exists twice —
`DELETE /records/{id}` next to `@DeleteMapping("/delete/{id}")` (w1-03 pain
1+3: 5× `@DeleteMapping("/delete")`, `/update/draft`, `/update/submit`).

## What fires (exact tokens only)

- Kebab or camel tokens matching a CRUD verb: `delete`, `update`, `get`,
  `create`, `save`, `find`, … — including inside one segment
  (`/getUserById` → `get`).
- The suggestion names the HTTP shape that replaces it (e.g. `PUT
  /<collection>/{id}` for `update`).

## What stays silent (FP shield)

- Substrings and derived nouns: `getter`, `updates`, `preview`,
  `rebuild` — token-boundary matching only.
- Non-CRUD action verbs (`search`, `export`, `generate`, `checkout`) — they
  are R2xx-05 `custom-method-post`'s subject.

## Spring example — compliant

```java
// The HTTP verb IS the action; the path names only resources.
@DeleteMapping("/records/{id}")
public void remove(@PathVariable Long id) { ... }
```

## Spring example — violation

```java
// The verb is stated twice — the transport AND the path.
@GetMapping("/find-by-id/{id}")
public BookDto findBookById(@PathVariable Long id) { ... }
```

## Suppression

```java
// vanguard:ignore R1xx-02 <reason>
```

Path-scoped suppression lives in `.vanguard.yaml` (`suppressions:` with a
`reason`, §6.4.1) when the URL is a published contract you cannot rename.

## Scope notes

- The verb set is a fixed CRUD lexicon (`rules/resources.go
  crudVerbTokens`); non-CRUD verbs never fire here.
- One aspect per rule: a segment carrying a verb is skipped by R1xx-01
  (no double reporting) but is judged by this rule AND R1xx-03 (route
  shape) — two aspects of one defect.
